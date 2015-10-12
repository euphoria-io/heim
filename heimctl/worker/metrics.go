package worker

import "github.com/prometheus/client_golang/prometheus"

var (
	claimedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "claimed",
		Subsystem: "jobs",
		Help:      "Number of claimed jobs per queue",
	}, []string{"queue"})

	dueGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "due",
		Subsystem: "jobs",
		Help:      "Number of past-due jobs per queue",
	}, []string{"queue"})

	waitingGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "waiting",
		Subsystem: "jobs",
		Help:      "Number of waiting jobs per queue",
	}, []string{"queue"})
)

func init() {
	prometheus.MustRegister(claimedGauge)
	prometheus.MustRegister(dueGauge)
	prometheus.MustRegister(waitingGauge)
}
