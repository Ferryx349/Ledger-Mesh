package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ledgerqueue_jobs_processed_total",
		Help: "Jobs completed by type and outcome",
	}, []string{"type", "outcome"})

	JobsRetried = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledgerqueue_jobs_retried_total",
		Help: "Jobs requeued for retry",
	})

	JobsDeadLettered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledgerqueue_jobs_dead_lettered_total",
		Help: "Jobs moved to dead-letter queue",
	})

	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledgerqueue_queue_depth",
		Help: "Main queue length",
	})

	DLQDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ledgerqueue_dlq_depth",
		Help: "Dead-letter queue length",
	})

	WebhookIngested = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledgerqueue_webhooks_ingested_total",
		Help: "Payment webhooks accepted",
	})

	ReconciliationRuns = promauto.NewCounter(prometheus.CounterOpts{
		Name: "ledgerqueue_reconciliation_runs_total",
		Help: "Reconciliation runs completed",
	})
)
