package activity

import "github.com/prometheus/client_golang/prometheus"

var (
	bounceActivity = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "bounces",
		Subsystem: "activity",
		Help:      "Number of bounces per room",
	}, []string{"room"})

	joinActivity = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "joins",
		Subsystem: "activity",
		Help:      "Number of joins per room",
	}, []string{"room"})

	partActivity = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "parts",
		Subsystem: "activity",
		Help:      "Number of parts per room",
	}, []string{"room"})

	messageActivity = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "messages",
		Subsystem: "activity",
		Help:      "Number of messages per room",
	}, []string{"room"})
)

func init() {
	prometheus.MustRegister(bounceActivity)
	prometheus.MustRegister(joinActivity)
	prometheus.MustRegister(partActivity)
	prometheus.MustRegister(messageActivity)
}
