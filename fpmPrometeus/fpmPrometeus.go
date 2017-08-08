package fpmPrometeus

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/tomasen/fcgi_client"

	"github.com/code-tool/docker-fpm-wrapper/fpmConfig"
)

const namespace = "phpfpm"

type Status struct {
	Name               string `json:"pool"`
	ProcessManager     string `json:"process manager"`
	StartTime          int    `json:"start time"`
	StartSince         int    `json:"start since"`
	AcceptedConn       int    `json:"accepted conn"`
	ListenQueue        int    `json:"listen queue"`
	MaxListenQueue     int    `json:"max listen queue"`
	ListenQueueLen     int    `json:"listen queue len"`
	IdleProcesses      int    `json:"idle processes"`
	ActiveProcesses    int    `json:"active processes"`
	TotalProcesses     int    `json:"total processes"`
	MaxActiveProcesses int    `json:"max active processes"`
	MaxChildrenReached int    `json:"max children reached"`
	SlowRequests       int    `json:"slow requests"`
}

type stat struct {
	mu       *sync.Mutex
	statuses []Status
	pools    []fpmConfig.Pool
}

func Register(fpmConfigPath string, update time.Duration) error {
	cfg, err := fpmConfig.Parse(fpmConfigPath)
	if err != nil {
		return err
	}
	fmpStatus := NewFPMPoolStatus(cfg.Pools)
	prometheus.MustRegister(fmpStatus)

	go func() {
		for {
			go fmpStatus.stat.UpdateStatuses()
			time.Sleep(update)
		}
	}()

	return nil
}

func (s *stat) GetStatuses() []Status {
	s.mu.Lock()
	statuses := make([]Status, len(s.statuses))
	copy(statuses, s.statuses)
	s.mu.Unlock()
	return statuses
}

func (s *stat) UpdateStatuses() error {
	statusCh := make(chan Status, 1)
	errCh := make(chan error, 1)
	for _, pool := range s.pools {
		go getStats(pool.Listen, pool.StatusPath, statusCh, errCh)
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

	startSince         *prometheus.CounterVec
	acceptedConn       *prometheus.CounterVec
	maxListenQueue     *prometheus.CounterVec
	maxActiveProcesses *prometheus.CounterVec
	maxChildrenReached *prometheus.CounterVec
	slowRequests       *prometheus.CounterVec
}

func (e *FPMPoolStatus) resetMetrics() {
	e.listenQueue.Reset()
	e.listenQueueLen.Reset()
	e.idleProcesses.Reset()
	e.activeProcesses.Reset()
	e.totalProcesses.Reset()
	e.startSince.Reset()
	e.acceptedConn.Reset()
	e.maxListenQueue.Reset()
	e.maxActiveProcesses.Reset()
	e.maxChildrenReached.Reset()
	e.slowRequests.Reset()
}

func NewFPMPoolStatus(pools []fpmConfig.Pool) *FPMPoolStatus {
	poolLabelNames := []string{"pool_name"}

	return &FPMPoolStatus{
		stat: stat{
			mu:       &sync.Mutex{},
			pools:    pools,
			statuses: make([]Status, len(pools)),
		},
		startSince: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "start_since",
				Help:      "Number of seconds since FPM has started",
			},
			poolLabelNames,
		),
		acceptedConn: prometheus.NewCounterVec(
			prometheus.CounterOpts{
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
		maxListenQueue: prometheus.NewCounterVec(
			prometheus.CounterOpts{
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
		maxActiveProcesses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "max_active_processes",
				Help:      "The maximum number of active processes since FPM has started",
			},
			poolLabelNames,
		),
		maxChildrenReached: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "max_children_reached",
				Help:      "The number of times, the process limit has been reached, when pm tries to start more children (works only for pm 'dynamic' and 'ondemand')",
			},
			poolLabelNames,
		),
		slowRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "slow_requests",
				Help:      "The number of requests that exceeded your request_slowlog_timeout value",
			},
			poolLabelNames,
		),
	}
}

func (e *FPMPoolStatus) reset() {
	e.listenQueue.Reset()
	e.listenQueueLen.Reset()
	e.idleProcesses.Reset()
	e.activeProcesses.Reset()
	e.totalProcesses.Reset()
	e.startSince.Reset()
	e.acceptedConn.Reset()
	e.maxListenQueue.Reset()
	e.maxActiveProcesses.Reset()
	e.maxChildrenReached.Reset()
	e.slowRequests.Reset()
}

func (e *FPMPoolStatus) Collect(ch chan<- prometheus.Metric) {
	e.resetMetrics()
	statuses := e.stat.GetStatuses()
	for _, p := range statuses {
		e.listenQueue.WithLabelValues(p.Name).Set(float64(p.ListenQueue))
		e.listenQueueLen.WithLabelValues(p.Name).Set(float64(p.ListenQueueLen))
		e.idleProcesses.WithLabelValues(p.Name).Set(float64(p.IdleProcesses))
		e.activeProcesses.WithLabelValues(p.Name).Set(float64(p.ActiveProcesses))
		e.totalProcesses.WithLabelValues(p.Name).Set(float64(p.TotalProcesses))
		e.startSince.WithLabelValues(p.Name).Add(float64(p.StartSince))
		e.acceptedConn.WithLabelValues(p.Name).Add(float64(p.AcceptedConn))
		e.maxListenQueue.WithLabelValues(p.Name).Add(float64(p.MaxListenQueue))
		e.maxActiveProcesses.WithLabelValues(p.Name).Add(float64(p.MaxActiveProcesses))
		e.maxChildrenReached.WithLabelValues(p.Name).Add(float64(p.MaxChildrenReached))
		e.slowRequests.WithLabelValues(p.Name).Add(float64(p.SlowRequests))

		e.listenQueue.WithLabelValues(p.Name).Collect(ch)
		e.listenQueueLen.WithLabelValues(p.Name).Collect(ch)
		e.idleProcesses.WithLabelValues(p.Name).Collect(ch)
		e.activeProcesses.WithLabelValues(p.Name).Collect(ch)
		e.totalProcesses.WithLabelValues(p.Name).Collect(ch)
		e.startSince.WithLabelValues(p.Name).Collect(ch)
		e.acceptedConn.WithLabelValues(p.Name).Collect(ch)
		e.maxListenQueue.WithLabelValues(p.Name).Collect(ch)
		e.maxActiveProcesses.WithLabelValues(p.Name).Collect(ch)
		e.maxChildrenReached.WithLabelValues(p.Name).Collect(ch)
		e.slowRequests.WithLabelValues(p.Name).Collect(ch)
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

func getStats(socket, script string, statusCh chan Status, errCh chan error) {
	if socket == "" {
		return
	}

	env := map[string]string{
		"QUERY_STRING":    "json&full",
		"SCRIPT_FILENAME": script,
		"SCRIPT_NAME":     script,
	}

	fcgi, err := fcgiclient.Dial("unix", socket)
	if err != nil {
		errCh <- err
		return
	}
	defer fcgi.Close()

	resp, err := fcgi.Get(env)
	if err != nil && err != io.EOF {
		errCh <- err
		return
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil && err != io.EOF {
		errCh <- err
		return
	}
	s := Status{}
	err = json.Unmarshal(content, &s)
	if err != nil {
		errCh <- err
		return
	}

	statusCh <- s
}
