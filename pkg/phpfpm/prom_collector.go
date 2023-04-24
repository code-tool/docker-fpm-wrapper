package phpfpm

import (
	"strings"
	"sync"
	"unicode"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type PromCollector struct {
	log     *zap.Logger
	metrics *PromMetrics
	pools   []Pool
}

func NewPromCollector(log *zap.Logger, metrics *PromMetrics, pools []Pool) *PromCollector {
	return &PromCollector{
		log:     log,
		metrics: metrics,
		pools:   pools,
	}
}

func (c *PromCollector) Describe(descs chan<- *prometheus.Desc) {
	c.metrics.ListenQueue.Describe(descs)
	c.metrics.ListenQueueLen.Describe(descs)
	c.metrics.IdleProcesses.Describe(descs)
	c.metrics.ActiveProcesses.Describe(descs)
	c.metrics.TotalProcesses.Describe(descs)
	c.metrics.AcceptedConn.Describe(descs)

	c.metrics.StartSince.Describe(descs)
	c.metrics.MaxListenQueue.Describe(descs)
	c.metrics.MaxActiveProcesses.Describe(descs)
	c.metrics.MaxChildrenReached.Describe(descs)
	c.metrics.SlowRequests.Describe(descs)
}

func (c *PromCollector) setAndCollect(gaugeVec *prometheus.GaugeVec, poolName string, val int, ch chan<- prometheus.Metric) {
	gauge := gaugeVec.WithLabelValues(poolName)
	gauge.Set(float64(val))
	gauge.Collect(ch)
}

func (c *PromCollector) getStatusListenPath(pool Pool) string {
	if pool.StatusListen != "" {
		return pool.StatusListen
	}

	return pool.Listen
}

func (c *PromCollector) isDigitOnlyStr(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}

	return true
}

func (c *PromCollector) listenToNetAndAddr(listen string) (string, string) {
	network := "unix"
	const localhost = "127.0.0.1"
	semicolonPos := strings.IndexByte(listen, ':')

	if semicolonPos != -1 {
		network = "tcp"
		if semicolonPos == 0 {
			listen = localhost + listen
		}
	}

	if c.isDigitOnlyStr(listen) {
		network = "tcp"
		listen = localhost + ":" + listen
	}

	return network, listen
}

func (c *PromCollector) collectForPool(pool Pool, ch chan<- prometheus.Metric) {
	net, addr := c.listenToNetAndAddr(c.getStatusListenPath(pool))
	status, err := GetStats(net, addr, pool.StatusPath)
	if err != nil {
		c.log.Error("can't collect metrics", zap.String("pool", pool.Name), zap.Error(err))
		return
	}

	c.setAndCollect(c.metrics.ListenQueue, status.Name, status.ListenQueue, ch)
	c.setAndCollect(c.metrics.ListenQueueLen, status.Name, status.ListenQueueLen, ch)
	c.setAndCollect(c.metrics.IdleProcesses, status.Name, status.IdleProcesses, ch)
	c.setAndCollect(c.metrics.ActiveProcesses, status.Name, status.ActiveProcesses, ch)
	c.setAndCollect(c.metrics.TotalProcesses, status.Name, status.TotalProcesses, ch)
	c.setAndCollect(c.metrics.AcceptedConn, status.Name, status.AcceptedConn, ch)
	c.setAndCollect(c.metrics.StartSince, status.Name, status.StartSince, ch)
	c.setAndCollect(c.metrics.MaxListenQueue, status.Name, status.MaxListenQueue, ch)
	c.setAndCollect(c.metrics.MaxActiveProcesses, status.Name, status.MaxActiveProcesses, ch)
	c.setAndCollect(c.metrics.MaxChildrenReached, status.Name, status.MaxChildrenReached, ch)
	c.setAndCollect(c.metrics.SlowRequests, status.Name, status.SlowRequests, ch)
}

func (c *PromCollector) Collect(metrics chan<- prometheus.Metric) {
	var wg sync.WaitGroup
	for pIdx := range c.pools {
		pool := c.pools[pIdx]

		if pool.StatusPath == "" {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()

			c.collectForPool(pool, metrics)
		}()
	}

	wg.Wait()
}
