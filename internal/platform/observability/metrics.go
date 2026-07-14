package observability

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var defaultDurationBuckets = []float64{
	0.005,
	0.01,
	0.025,
	0.05,
	0.1,
	0.25,
	0.5,
	1,
	2.5,
	5,
	10,
}

type Metrics struct {
	registry      *prometheus.Registry
	requests      *prometheus.CounterVec
	duration      *prometheus.HistogramVec
	responseSize  *prometheus.HistogramVec
	requestsInFly prometheus.Gauge
}

func NewMetrics(service string) *Metrics {
	service = strings.TrimSpace(service)
	constantLabels := prometheus.Labels{"service": service}

	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		registry: registry,
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace:   "edgeguard",
			Name:        "http_requests_total",
			Help:        "Total number of HTTP requests processed by EdgeGuard services.",
			ConstLabels: constantLabels,
		}, []string{"method", "route", "status_code"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:   "edgeguard",
			Name:        "http_request_duration_seconds",
			Help:        "Duration of HTTP requests processed by EdgeGuard services.",
			ConstLabels: constantLabels,
			Buckets:     defaultDurationBuckets,
		}, []string{"method", "route"}),
		responseSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace:   "edgeguard",
			Name:        "http_response_size_bytes",
			Help:        "Size of HTTP responses returned by EdgeGuard services.",
			ConstLabels: constantLabels,
			Buckets:     prometheus.ExponentialBuckets(128, 2, 16),
		}, []string{"method", "route"}),
		requestsInFly: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   "edgeguard",
			Name:        "http_requests_in_flight",
			Help:        "Current number of in-flight HTTP requests.",
			ConstLabels: constantLabels,
		}),
	}

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		metrics.requests,
		metrics.duration,
		metrics.responseSize,
		metrics.requestsInFly,
	)

	return metrics
}

func (m *Metrics) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			request := c.Request()
			if isObservabilityPath(request.URL.Path) {
				return next(c)
			}

			m.requestsInFly.Inc()
			defer m.requestsInFly.Dec()

			startedAt := time.Now()
			originalResponse := c.Response()
			recorder := newResponseRecorder(originalResponse)
			c.SetResponse(recorder)
			defer c.SetResponse(originalResponse)

			err := next(c)
			status := recorder.status
			if err != nil && status < http.StatusBadRequest {
				status = http.StatusInternalServerError
			}

			route := strings.TrimSpace(c.Path())
			if route == "" {
				route = "unmatched"
			}

			method := request.Method
			m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
			m.duration.WithLabelValues(method, route).Observe(time.Since(startedAt).Seconds())
			m.responseSize.WithLabelValues(method, route).Observe(float64(recorder.bytes))

			return err
		}
	}
}

func (m *Metrics) Handler() echo.HandlerFunc {
	return echo.WrapHandler(promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	}))
}

func (m *Metrics) Gatherer() prometheus.Gatherer {
	return m.registry
}

func isObservabilityPath(path string) bool {
	switch path {
	case "/health", "/ready", "/metrics":
		return true
	default:
		return false
	}
}
