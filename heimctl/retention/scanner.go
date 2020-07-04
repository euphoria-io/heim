package retention // import "euphoria.io/heim/heimctl/retention"

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/go-gorp/gorp"
	"github.com/prometheus/client_golang/prometheus"

	"euphoria.io/heim/backend/psql"
	"euphoria.io/heim/cluster"
	"euphoria.io/scope"
)

const (
	maxErrors   = 3
	GracePeriod = time.Hour
)

func scanForExpired(ctx scope.Context, c cluster.Cluster, pb *psql.Backend) error {
	rows, err := pb.DbMap.Select(
		psql.Room{},
		"SELECT name, founded_by, retention_days FROM room WHERE retention_days > 0")
	if err == sql.ErrNoRows {
		lastExpiredScan.Set(float64(time.Now().Unix()))
		return nil
	}
	if err != nil {
		return err
	}
	for _, row := range rows {
		room, ok := row.(*psql.Room)
		if !ok {
			fmt.Printf("error: expected row of type *psql.Room, got %T\n", row)
			continue
		}
		var oldestRow struct {
			Oldest gorp.NullTime
		}
		err := pb.DbMap.SelectOne(&oldestRow,
			"SELECT Min(posted) AS oldest FROM message WHERE room = $1",
			room.Name)
		if err != nil {
			fmt.Printf("error selecting oldest message: %s\n", err)
			continue
		}
		if !oldestRow.Oldest.Valid {
			roomHasExpiredMsg.With(prometheus.Labels{"room": room.Name}).Set(0)
			continue
		}
		threshold := time.Now().Add(time.Duration(-room.RetentionDays)*24*time.Hour - GracePeriod)
		if oldestRow.Oldest.Time.Before(threshold) {
			roomHasExpiredMsg.With(prometheus.Labels{"room": room.Name}).Set(1)
		} else {
			roomHasExpiredMsg.With(prometheus.Labels{"room": room.Name}).Set(0)
		}
		lastExpiredScan.Set(float64(time.Now().Unix()))
	}
	return nil
}

func ExpiredScanLoop(ctx scope.Context, c cluster.Cluster, pb *psql.Backend, interval time.Duration) {
	defer ctx.WaitGroup().Done()

	errCount := 0
	for {
		t := time.After(interval)
		select {
		case <-ctx.Done():
			return
		case <-t:
			if err := scanForExpired(ctx, c, pb); err != nil {
				errCount++
				fmt.Printf("scan error [%d/%d]: %s", errCount, maxErrors, err)
				if errCount > maxErrors {
					fmt.Println("maximum scan errors exceeded, terminating")
					ctx.Terminate(fmt.Errorf("maximum scan errors exceeded"))
					return
				}
				continue
			}
			errCount = 0
		}
	}
}

func scanToDelete(ctx scope.Context, c cluster.Cluster, pb *psql.Backend) error {
	rows, err := pb.DbMap.Select(
		psql.Room{},
		"SELECT name, founded_by, retention_days FROM room WHERE retention_days > 0")
	if err == sql.ErrNoRows {
		lastDeleteScan.Set(float64(time.Now().Unix()))
		return nil
	}
	if err != nil {
		return err
	}
	for _, row := range rows {
		room, ok := row.(*psql.Room)
		if !ok {
			fmt.Printf("error: expected row of type *psql.Room, got %T\n", row)
			continue
		}
		// don't use grace period here- delete as soon as they expire
		threshold := time.Now().Add(time.Duration(-room.RetentionDays) * 24 * time.Hour)
		_, err := pb.DbMap.Exec(
			"DELETE FROM message WHERE room = $1 AND posted < $2",
			room.Name,
			threshold,
		)
		if err != nil && err != sql.ErrNoRows {
			fmt.Printf("error deleting rows: %s\n", err)
		}
		lastDeleteScan.Set(float64(time.Now().Unix()))
	}
	return nil
}

func DeleteScanLoop(ctx scope.Context, c cluster.Cluster, pb *psql.Backend, interval time.Duration) {
	defer ctx.WaitGroup().Done()

	errCount := 0
	for {
		t := time.After(interval)
		select {
		case <-ctx.Done():
			return
		case <-t:
			if err := scanToDelete(ctx, c, pb); err != nil {
				errCount++
				fmt.Printf("delete scan error [%d/%d]: %s", errCount, maxErrors, err)
				if errCount > maxErrors {
					fmt.Println("maximum delete scan errors exceeded, terminating")
					ctx.Terminate(fmt.Errorf("maximum delete scan errors exceeded"))
					return
				}
				continue
			}
			errCount = 0
		}
	}
}
