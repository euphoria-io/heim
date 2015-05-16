package presence

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/backend/psql"
	"euphoria.io/scope"
)

const (
	maxErrors = 3
	chunkSize = 1000
)

func ScanLoop(ctx scope.Context, c cluster.Cluster, pb *psql.Backend, interval time.Duration) {
	defer ctx.WaitGroup().Done()

	errCount := 0
	for {
		t := time.After(interval)
		select {
		case <-ctx.Done():
			return
		case <-t:
			if err := scan(ctx.Fork(), c, pb); err != nil {
				errCount++
				fmt.Printf("scan error [%d/%d]: %s", errCount, maxErrors, err)
				if errCount > maxErrors {
					fmt.Printf("maximum scan errors exceeded, terminating\n")
					ctx.Terminate(fmt.Errorf("maximum scan errors exceeded"))
					return
				}
				continue
			}
			errCount = 0
		}
	}
}

func scan(ctx scope.Context, c cluster.Cluster, pb *psql.Backend) error {
	rows, err := pb.DbMap.Select(
		psql.Presence{},
		"SELECT room, session_id, server_id, server_era, updated, fact FROM presence")
	if err != nil {
		return err
	}

	peers := map[string]string{}
	for _, desc := range c.Peers() {
		peers[desc.ID] = desc.Era
	}

	activeRows := 0
	activeRowsPerRoom := map[string]int{}
	activeSessionsPerAgent := map[string]int{}
	lurkingSessionsPerAgent := map[string]int{}

	lurkingRows := 0
	lurkingRowsPerRoom := map[string]int{}

	for _, row := range rows {
		presence, ok := row.(*psql.Presence)
		if !ok {
			fmt.Printf("error: expected row of type *psql.Presence, got %T\n", row)
			continue
		}

		if peers[presence.ServerID] == presence.ServerEra {
			activeRows++
			activeRowsPerRoom[presence.Room]++

			parts := strings.Split(presence.SessionID, "-")
			activeSessionsPerAgent[parts[0]]++

			// Check lurker status. Currently this is indicated by a blank name on the session.
			session, err := presence.SessionView()
			if err != nil {
				fmt.Printf("error: failed to extract session from presence row: %s\n", err)
				continue
			}
			if session.Name == "" {
				lurkingRows++
				lurkingRowsPerRoom[presence.Room]++
				lurkingSessionsPerAgent[parts[0]]++
			}
		}
	}

	rowCount.Set(float64(len(rows)))
	activeRowCount.Set(float64(activeRows))
	lurkingRowCount.Set(float64(lurkingRows))
	uniqueAgentCount.Set(float64(len(activeSessionsPerAgent)))
	uniqueLurkingAgentCount.Set(float64(len(lurkingSessionsPerAgent)))

	for room, count := range activeRowsPerRoom {
		activeRowCountPerRoom.With(prometheus.Labels{"room": room}).Set(float64(count))
	}

	for room, count := range lurkingRowsPerRoom {
		lurkingRowCountPerRoom.With(prometheus.Labels{"room": room}).Set(float64(count))
	}

	for _, count := range activeSessionsPerAgent {
		sessionsPerAgent.Observe(float64(count))
	}

	return nil
}
