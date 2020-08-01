package fetcher

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	SignaturesTotalMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "petitions_signatures",
			Help: "Number of signatures for a petition",
		},
		[]string{"url", "id", "action", "opened_at", "petition_url"},
	)

	FetcherErrorsTotalMetric = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "petitions_fetcher_errors_total",
			Help: "Number of errors encountered by the petitions fetcher",
		},
		[]string{"url"},
	)

	FetcherFetchesMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "petitions_fetcher_fetches",
			Help: "Number of fetches performed by the petitions fetcher",
		},
		[]string{"url"},
	)
)

func initMetrics() {
	prometheus.MustRegister(SignaturesTotalMetric)
	prometheus.MustRegister(FetcherErrorsTotalMetric)
	prometheus.MustRegister(FetcherFetchesMetric)
}
