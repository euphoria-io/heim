package presence

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"euphoria.io/heim/backend/psql"
	"euphoria.io/heim/cluster"
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
	type PresenceWithUserAgent struct {
		psql.Presence
		UserAgent string `db:"user_agent"`
	}

	rows, err := pb.DbMap.Select(
		PresenceWithUserAgent{},
		"SELECT p.room, p.session_id, p.server_id, p.server_era, p.updated, p.fact, s.user_agent"+
			" FROM presence p, session_log s WHERE p.session_id = s.session_id")
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
	webSessionsPerAgent := map[string]int{}

	lurkingRows := 0
	lurkingRowsPerRoom := map[string]int{}

	for _, row := range rows {
		presence, ok := row.(*PresenceWithUserAgent)
		if !ok {
			fmt.Printf("error: expected row of type *PresenceWithUserAgent, got %T\n", row)
			continue
		}

		if peers[presence.ServerID] == presence.ServerEra {
			activeRows++
			activeRowsPerRoom[presence.Room]++

			parts := strings.Split(presence.SessionID, "-")
			activeSessionsPerAgent[parts[0]]++

			// Check web-client status.
			// TODO: use positive fingerprint from web client instead of user-agent
			if presence.UserAgent != "" && !strings.HasPrefix(presence.UserAgent, "Python") {
				webSessionsPerAgent[parts[0]]++
			}

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
	uniqueWebAgentCount.Set(float64(len(webSessionsPerAgent)))

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
