package phpfpm

import (
	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "phpfpm"

type PromMetrics struct {
	ListenQueue     *prometheus.GaugeVec
	ListenQueueLen  *prometheus.GaugeVec
	IdleProcesses   *prometheus.GaugeVec
	ActiveProcesses *prometheus.GaugeVec
	TotalProcesses  *prometheus.GaugeVec
	AcceptedConn    *prometheus.GaugeVec

	StartSince         *prometheus.GaugeVec
	MaxListenQueue     *prometheus.GaugeVec
	MaxActiveProcesses *prometheus.GaugeVec
	MaxChildrenReached *prometheus.GaugeVec
	SlowRequests       *prometheus.GaugeVec
}

func NewPromMetrics() *PromMetrics {
	poolLabelNames := []string{"pool_name"}

	return &PromMetrics{
		StartSince: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "start_since",
				Help:      "Number of seconds since FPM has started",
			},
			poolLabelNames,
		),
		AcceptedConn: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "accepted_conn",
				Help:      "The number of requests accepted by the pool",
			},
			poolLabelNames,
		),
		ListenQueue: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "listen_queue",
				Help:      "The number of requests in the queue of pending connections",
			},
			poolLabelNames,
		),
		MaxListenQueue: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "max_listen_queue",
				Help:      "The maximum number of requests in the queue of pending connections since FPM has started",
			},
			poolLabelNames,
		),
		ListenQueueLen: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "listen_queue_len",
				Help:      "The size of the socket queue of pending connections",
			},
			poolLabelNames,
		),
		IdleProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "idle_processes",
				Help:      "The number of idle processes",
			},
			poolLabelNames,
		),
		ActiveProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_processes",
				Help:      "The number of active processes",
			},
			poolLabelNames,
		),
		TotalProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "total_processes",
				Help:      "The number of idle + active processes",
			},
			poolLabelNames,
		),
		MaxActiveProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "max_active_processes",
				Help:      "The maximum number of active processes since FPM has started",
			},
			poolLabelNames,
		),
		MaxChildrenReached: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "max_children_reached",
				Help:      "The number of times, the process limit has been reached, when pm tries to start more children (works only for pm 'dynamic' and 'ondemand')",
			},
			poolLabelNames,
		),
		SlowRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "slow_requests",
				Help:      "The number of requests that exceeded your request_slowlog_timeout value",
			},
			poolLabelNames,
		),
	}
}
