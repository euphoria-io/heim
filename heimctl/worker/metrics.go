package worker

import "github.com/prometheus/client_golang/prometheus"

var (
	claimedGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "claimed",
		Subsystem: "jobs",
		Help:      "Number of claimed jobs per queue",
	}, []string{"queue"})

	completedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "completed",
		Subsystem: "jobs",
		Help:      "Counter of job claims completed by this worker.",
	}, []string{"queue"})

	dueGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "due",
		Subsystem: "jobs",
		Help:      "Number of past-due jobs per queue",
	}, []string{"queue"})

	errorCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name:      "error",
		Subsystem: "jobs",
		Help:      "Counter of system errors with job management under this worker.",
	})

	failedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "failed",
		Subsystem: "jobs",
		Help:      "Counter of job claims failed by this worker.",
	}, []string{"queue"})

	processedCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name:      "processed",
		Subsystem: "jobs",
		Help:      "Counter of job claims processed by this worker.",
	}, []string{"queue"})

	waitingGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name:      "waiting",
		Subsystem: "jobs",
		Help:      "Number of waiting jobs per queue",
	}, []string{"queue"})
)

func init() {
	prometheus.MustRegister(claimedGauge)
	prometheus.MustRegister(completedCounter)
	prometheus.MustRegister(dueGauge)
	prometheus.MustRegister(errorCounter)
	prometheus.MustRegister(failedCounter)
	prometheus.MustRegister(processedCounter)
	prometheus.MustRegister(waitingGauge)
}
