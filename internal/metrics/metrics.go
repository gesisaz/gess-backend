package metrics

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"gess-backend/database"
	"gess-backend/mail"
	"gess-backend/mpesa"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "route", "status_class"},
	)
	dbUp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_up",
		Help: "1 if PostgreSQL ping succeeded, else 0",
	})
	mailConfigured = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mail_configured",
		Help: "1 if RESEND_API_KEY is set",
	})
	mpesaConsumerConfigured = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mpesa_consumer_configured",
		Help: "1 if M-PESA OAuth consumer credentials are set",
	})
)

// MetricsHandler serves Prometheus metrics on /metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// ObserveRequest records HTTP latency for Prometheus.
func ObserveRequest(method, route, statusClass string, d time.Duration) {
	httpDuration.WithLabelValues(method, route, statusClass).Observe(d.Seconds())
}

// RegisterIntegrationGauges sets static configuration gauges once at startup.
func RegisterIntegrationGauges() {
	if mail.Configured() {
		mailConfigured.Set(1)
	} else {
		mailConfigured.Set(0)
	}
	if mpesa.ConsumerConfigured() {
		mpesaConsumerConfigured.Set(1)
	} else {
		mpesaConsumerConfigured.Set(0)
	}
}

// StartDBPingLoop updates db_up periodically until ctx is cancelled.
func StartDBPingLoop(ctx context.Context, interval time.Duration) {
	go func() {
		tick := func() {
			pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			err := database.Ping(pingCtx)
			cancel()
			if err != nil {
				dbUp.Set(0)
				slog.Error("database ping failed", "err", err)
				return
			}
			dbUp.Set(1)
		}
		tick()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				tick()
			}
		}
	}()
}
