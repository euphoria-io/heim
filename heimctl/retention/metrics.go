package retention

import "github.com/prometheus/client_golang/prometheus"

var (
	roomHasExpiredMsg = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "room_has_expired",
		Subsystem: "retention",
		Help:      "Whether a room has expired messages, labeled by room name.",
	}, []string{"room"})

	lastExpiredScan = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "last_expired_scan",
		Subsystem: "retention",
		Help:      "The last Unix time the expired message scanner loop completed.",
	})

	lastDeleteScan = prometheus.NewGauge(prometheus.GaugeOpts{
		Name:      "last_delete_scan",
		Subsystem: "retention",
		Help:      "The last Unix time the delete message scanner loop completed.",
	})
)

func init() {
	prometheus.MustRegister(roomHasExpiredMsg)
	prometheus.MustRegister(lastExpiredScan)
	prometheus.MustRegister(lastDeleteScan)
}
