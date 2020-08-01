package fetcher

import (
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/tlwr/petitions-exporter/pkg/petitions-client"
)

func init() {
	initMetrics()
}

type Fetcher interface {
	Start()
	Stop()
	Wait()
}

type fetcher struct {
	baseURL  string
	client   client.Client
	logger   *logrus.Logger
	interval time.Duration

	wg   sync.WaitGroup
	stop chan struct{}
}

func New(baseURL string, interval time.Duration, logger *logrus.Logger) Fetcher {
	c := client.New(baseURL, logger)

	f := &fetcher{
		baseURL:  baseURL,
		client:   c,
		logger:   logger,
		interval: interval,

		stop: make(chan struct{}),
	}

	return f
}

func (f *fetcher) fetch() {
	start := time.Now()

	f.logger.Info("fetch-list")
	petitions, err := f.client.List()

	if err != nil {
		f.logger.Error(err)
		FetcherErrorsTotalMetric.With(
			prometheus.Labels{"url": f.baseURL},
		).Inc()
	} else {
		for _, petition := range petitions {
			SignaturesTotalMetric.With(prometheus.Labels{
				"url":          f.baseURL,
				"petition_url": petition.URL(),
				"id":           fmt.Sprintf("%d", petition.ID()),
				"action":       petition.Action(),
				"opened_at":    petition.OpenedAt().Format(time.RFC3339),
			}).Set(float64(petition.SignatureCount()))
		}
	}

	delta := time.Now().Sub(start)
	FetcherFetchesMetric.With(
		prometheus.Labels{"url": f.baseURL},
	).Observe(delta.Seconds())
}

func (f *fetcher) Start() {
	f.wg.Add(1)

	go func() {
		f.fetch()

		for {
			f.logger.Info("fetch-loop")
			select {
			case <-f.stop:
				f.logger.Info("fetch-stop")
				break // from select
			case <-time.After(f.interval):
				f.fetch()
				continue // next loop iteration
			}

			break // from break in select
		}

		f.wg.Done()
	}()
}

func (f *fetcher) Stop() {
	f.stop <- struct{}{}
}

func (f *fetcher) Wait() {
	f.wg.Wait()
}
