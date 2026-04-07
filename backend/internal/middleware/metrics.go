package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type Metrics struct {
	mu              sync.RWMutex
	totalRequests   atomic.Int64
	totalErrors     atomic.Int64
	activeRequests  atomic.Int64
	statusCounts    map[int]*atomic.Int64
	methodCounts    map[string]*atomic.Int64
	totalLatencyMs  atomic.Int64
	startTime       time.Time
}

var globalMetrics = &Metrics{
	statusCounts: make(map[int]*atomic.Int64),
	methodCounts: make(map[string]*atomic.Int64),
	startTime:    time.Now(),
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		globalMetrics.activeRequests.Add(1)
		start := time.Now()

		ctx.Next()

		elapsed := time.Since(start).Milliseconds()
		globalMetrics.totalLatencyMs.Add(elapsed)
		globalMetrics.activeRequests.Add(-1)
		globalMetrics.totalRequests.Add(1)

		status := ctx.Writer.Status()
		if status >= 500 {
			globalMetrics.totalErrors.Add(1)
		}

		globalMetrics.getStatusCounter(status).Add(1)
		globalMetrics.getMethodCounter(ctx.Request.Method).Add(1)
	}
}

func MetricsHandler() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		m := globalMetrics
		uptime := time.Since(m.startTime).Seconds()
		total := m.totalRequests.Load()
		errCount := m.totalErrors.Load()
		active := m.activeRequests.Load()
		latency := m.totalLatencyMs.Load()

		var avgLatency float64
		if total > 0 {
			avgLatency = float64(latency) / float64(total)
		}

		body := fmt.Sprintf(
			"# HELP shiro_http_requests_total Total HTTP requests\n"+
				"# TYPE shiro_http_requests_total counter\n"+
				"shiro_http_requests_total %d\n"+
				"# HELP shiro_http_errors_total Total HTTP 5xx errors\n"+
				"# TYPE shiro_http_errors_total counter\n"+
				"shiro_http_errors_total %d\n"+
				"# HELP shiro_http_active_requests Active HTTP requests\n"+
				"# TYPE shiro_http_active_requests gauge\n"+
				"shiro_http_active_requests %d\n"+
				"# HELP shiro_http_avg_latency_ms Average request latency ms\n"+
				"# TYPE shiro_http_avg_latency_ms gauge\n"+
				"shiro_http_avg_latency_ms %.2f\n"+
				"# HELP shiro_uptime_seconds Server uptime seconds\n"+
				"# TYPE shiro_uptime_seconds gauge\n"+
				"shiro_uptime_seconds %.0f\n",
			total, errCount, active, avgLatency, uptime,
		)

		m.mu.RLock()
		for status, counter := range m.statusCounts {
			body += fmt.Sprintf("shiro_http_status{code=\"%d\"} %d\n", status, counter.Load())
		}
		for method, counter := range m.methodCounts {
			body += fmt.Sprintf("shiro_http_method{method=\"%s\"} %d\n", method, counter.Load())
		}
		m.mu.RUnlock()

		ctx.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
	}
}

func (m *Metrics) getStatusCounter(status int) *atomic.Int64 {
	m.mu.RLock()
	counter, ok := m.statusCounts[status]
	m.mu.RUnlock()
	if ok {
		return counter
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	counter, ok = m.statusCounts[status]
	if !ok {
		counter = &atomic.Int64{}
		m.statusCounts[status] = counter
	}
	return counter
}

func (m *Metrics) getMethodCounter(method string) *atomic.Int64 {
	m.mu.RLock()
	counter, ok := m.methodCounts[method]
	m.mu.RUnlock()
	if ok {
		return counter
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	counter, ok = m.methodCounts[method]
	if !ok {
		counter = &atomic.Int64{}
		m.methodCounts[method] = counter
	}
	return counter
}
