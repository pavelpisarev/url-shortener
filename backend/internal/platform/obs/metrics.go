package obs

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "route", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	redirectsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "redirects_total",
		Help: "Total number of successful redirects.",
	})
	cacheHitsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_hits_total",
		Help: "Total number of redirect cache hits.",
	})
	cacheMissesTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cache_misses_total",
		Help: "Total number of redirect cache misses.",
	})
	clicksEnqueuedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "clicks_enqueued_total",
		Help: "Total number of clicks enqueued for aggregation.",
	})
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		redirectsTotal,
		cacheHitsTotal,
		cacheMissesTotal,
		clicksEnqueuedTotal,
	)
}

func Handler() http.Handler {
	return promhttp.Handler()
}

func IncRedirect()      { redirectsTotal.Inc() }
func IncCacheHit()      { cacheHitsTotal.Inc() }
func IncCacheMiss()     { cacheMissesTotal.Inc() }
func IncClickEnqueued() { clicksEnqueuedTotal.Inc() }

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)

		route := r.Pattern
		if i := strings.IndexByte(route, ' '); i >= 0 {
			route = route[i+1:]
		}
		if route == "" {
			route = "unmatched"
		}

		httpRequestsTotal.WithLabelValues(r.Method, route, strconv.Itoa(rw.status)).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
