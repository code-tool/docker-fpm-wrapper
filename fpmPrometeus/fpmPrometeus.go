package fpmPrometeus

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/code-tool/docker-fpm-wrapper/fpmConfig"
	"github.com/code-tool/docker-fpm-wrapper/pkg/phpfpm"
)

const namespace = "phpfpm"

type stat struct {
	mu       *sync.Mutex
	statuses []phpfpm.Status
	pools    []fpmConfig.Pool
}

func Register(fpmConfigPath string, update time.Duration) error {
	cfg, err := fpmConfig.Parse(fpmConfigPath)
	if err != nil {
		return err
	}
	fpmStatus := NewFPMPoolStatus(cfg.Pools)
	prometheus.MustRegister(fpmStatus)

	go startUpdateStatuses(fpmStatus, update)
	return nil
}

func startUpdateStatuses(fpmStatus *FPMPoolStatus, update time.Duration) {
	var t time.Time
	for {
		t = time.Now()
		fpmStatus.stat.UpdateStatuses()
		sleep := update - time.Now().Sub(t)
		time.Sleep(sleep)
	}
}

func (s *stat) GetStatuses() []phpfpm.Status {
	s.mu.Lock()
	statuses := make([]phpfpm.Status, len(s.statuses))
	copy(statuses, s.statuses)
	s.mu.Unlock()
	return statuses
}

func (s *stat) UpdateStatuses() error {
	statusCh := make(chan phpfpm.Status, 1)
	errCh := make(chan error, 1)
	for _, pool := range s.pools {
		go func() {
			status, err := phpfpm.GetStats(pool.Listen, pool.StatusPath)
			if err != nil {
				errCh <- err
			} else {
				statusCh <- *status
			}
		}()
	}

	errors := []string{}

	s.mu.Lock()
	for i := range s.statuses {
		select {
		case err := <-errCh:
			errors = append(errors, err.Error())
		case s.statuses[i] = <-statusCh:
		}
	}
	s.mu.Unlock()

	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

type FPMPoolStatus struct {
	stat stat

	listenQueue     *prometheus.GaugeVec
	listenQueueLen  *prometheus.GaugeVec
	idleProcesses   *prometheus.GaugeVec
	activeProcesses *prometheus.GaugeVec
	totalProcesses  *prometheus.GaugeVec
	acceptedConn    *prometheus.GaugeVec

	startSince         *prometheus.GaugeVec
	maxListenQueue     *prometheus.GaugeVec
	maxActiveProcesses *prometheus.GaugeVec
	maxChildrenReached *prometheus.GaugeVec
	slowRequests       *prometheus.GaugeVec
}

func NewFPMPoolStatus(pools []fpmConfig.Pool) *FPMPoolStatus {
	poolLabelNames := []string{"pool_name"}

	return &FPMPoolStatus{
		stat: stat{
			mu:       &sync.Mutex{},
			pools:    pools,
			statuses: make([]phpfpm.Status, len(pools)),
		},
		startSince: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "start_since",
				Help:      "Number of seconds since FPM has started",
			},
			poolLabelNames,
		),
		acceptedConn: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "accepted_conn",
				Help:      "The number of requests accepted by the pool",
			},
			poolLabelNames,
		),
		listenQueue: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "listen_queue",
				Help:      "The number of requests in the queue of pending connections",
			},
			poolLabelNames,
		),
		maxListenQueue: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "max_listen_queue",
				Help:      "The maximum number of requests in the queue of pending connections since FPM has started",
			},
			poolLabelNames,
		),
		listenQueueLen: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "listen_queue_len",
				Help:      "The size of the socket queue of pending connections",
			},
			poolLabelNames,
		),
		idleProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "idle_processes",
				Help:      "The number of idle processes",
			},
			poolLabelNames,
		),
		activeProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_processes",
				Help:      "The number of active processes",
			},
			poolLabelNames,
		),
		totalProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "total_processes",
				Help:      "The number of idle + active processes",
			},
			poolLabelNames,
		),
		maxActiveProcesses: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "max_active_processes",
				Help:      "The maximum number of active processes since FPM has started",
			},
			poolLabelNames,
		),
		maxChildrenReached: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "max_children_reached",
				Help:      "The number of times, the process limit has been reached, when pm tries to start more children (works only for pm 'dynamic' and 'ondemand')",
			},
			poolLabelNames,
		),
		slowRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "slow_requests",
				Help:      "The number of requests that exceeded your request_slowlog_timeout value",
			},
			poolLabelNames,
		),
	}
}

func setAndCollect(gaugeVec *prometheus.GaugeVec, poolName string, val int, ch chan<- prometheus.Metric) {
	gauge := gaugeVec.WithLabelValues(poolName)
	gauge.Set(float64(val))
	gauge.Collect(ch)
}

func (e *FPMPoolStatus) Collect(ch chan<- prometheus.Metric) {
	statuses := e.stat.GetStatuses()
	for _, p := range statuses {
		setAndCollect(e.listenQueue, p.Name, p.ListenQueue, ch)
		setAndCollect(e.listenQueueLen, p.Name, p.ListenQueueLen, ch)
		setAndCollect(e.idleProcesses, p.Name, p.IdleProcesses, ch)
		setAndCollect(e.activeProcesses, p.Name, p.ActiveProcesses, ch)
		setAndCollect(e.totalProcesses, p.Name, p.TotalProcesses, ch)
		setAndCollect(e.acceptedConn, p.Name, p.AcceptedConn, ch)
		setAndCollect(e.startSince, p.Name, p.StartSince, ch)
		setAndCollect(e.maxListenQueue, p.Name, p.MaxListenQueue, ch)
		setAndCollect(e.maxActiveProcesses, p.Name, p.MaxActiveProcesses, ch)
		setAndCollect(e.maxChildrenReached, p.Name, p.MaxChildrenReached, ch)
		setAndCollect(e.slowRequests, p.Name, p.SlowRequests, ch)
	}
}

func (e *FPMPoolStatus) Describe(ch chan<- *prometheus.Desc) {
	e.listenQueue.Describe(ch)
	e.listenQueueLen.Describe(ch)
	e.idleProcesses.Describe(ch)
	e.activeProcesses.Describe(ch)
	e.totalProcesses.Describe(ch)
	e.startSince.Describe(ch)
	e.acceptedConn.Describe(ch)
	e.maxListenQueue.Describe(ch)
	e.maxActiveProcesses.Describe(ch)
	e.maxChildrenReached.Describe(ch)
	e.slowRequests.Describe(ch)
}
