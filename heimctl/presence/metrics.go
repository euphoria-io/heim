package presence

import "github.com/prometheus/client_golang/prometheus"

var (
	rowCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "total_rows",
		Subsystem: "presence",
		Help:      "Total size of presence table.",
	})

	activeRowCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "active_rows",
		Subsystem: "presence",
		Help:      "Number of active rows in the presence table.",
	})

	activeRowCountPerRoom = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "active_rows_per_room",
		Subsystem: "presence",
		Help:      "Number of active rows in the presence table, labelled by room.",
	}, []string{"room"})

	uniqueAgentCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "unique_agents",
		Subsystem: "presence",
		Help:      "Number of unique, active agents in the presence table.",
	})

	sessionsPerAgent = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:      "sessions_per_agent",
		Subsystem: "presence",
		Help:      "Number of simultaneous live sessions for each active agent.",
		Buckets:   prometheus.LinearBuckets(0, 1, 10),
	})
)

func init() {
	prometheus.MustRegister(rowCount)
	prometheus.MustRegister(activeRowCount)
	prometheus.MustRegister(activeRowCountPerRoom)
	prometheus.MustRegister(uniqueAgentCount)
	prometheus.MustRegister(sessionsPerAgent)
}
