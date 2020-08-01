package fetcher_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/prometheus/client_golang/prometheus"
	promtest "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"

	. "github.com/tlwr/petitions-exporter/pkg/petitions-fetcher"
)

func TestPetitionsFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PetitionsFetcher")
}

var _ = Describe("Fetcher", func() {
	var (
		server *ghttp.Server

		logger   *logrus.Logger
		baseURL  string
		interval time.Duration

		fetcher Fetcher
	)

	BeforeEach(func() {
		server = ghttp.NewServer()

		logger = logrus.New()
		logger.SetOutput(GinkgoWriter)
		logger.SetFormatter(&logrus.JSONFormatter{})

		baseURL = server.URL()

		interval = 15 * time.Millisecond

		fetcher = New(baseURL, interval, logger)
	})

	AfterEach(func() {
		server.Close()
	})

	Context("when the petitions is working", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/petitions.json", "state=open"),
					ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{
						"links": {
							"next": "%s/petitions.json?page=2&state=open"
						},
						"data": [{
							"id": 1,
							"attributes": {
								"action": "start a thing",
								"signature_count": 123,
								"opened_at": "2020-07-02T13:40:09.021Z"
							}
						},{
							"id": 2,
							"attributes": {
								"action": "stop a thing",
								"signature_count": 456,
								"opened_at": "2020-07-03T13:40:09.021Z"
							}
						}]
					}`, server.URL())),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/petitions.json", "page=2&state=open"),
					ghttp.RespondWith(http.StatusOK, `{
						"links": {},
						"data": [{
							"id": 3,
							"attributes": {
								"action": "change a thing",
								"signature_count": 789,
								"opened_at": "2020-07-04T13:40:09.021Z"
							}
						},{
							"id": 4,
							"attributes": {
								"action": "return a thing",
								"signature_count": 987,
								"opened_at": "2020-07-05T13:40:09.021Z"
							}
						}]
					}`),
				),
			)
		})

		It("should fetch some petitions", func() {
			fetcher.Start()

			Eventually(server.ReceivedRequests).Should(HaveLen(2))

			fetcher.Stop()
			fetcher.Wait()
		})

		It("should expose petitions as metrics", func() {
			errorMetricValueBefore := promtest.ToFloat64(
				FetcherErrorsTotalMetric.With(prometheus.Labels{"url": server.URL()}),
			)

			fetcher.Start()

			Eventually(server.ReceivedRequests).Should(HaveLen(2))

			fetcher.Stop()
			fetcher.Wait()

			errorMetricValueAfter := promtest.ToFloat64(
				FetcherErrorsTotalMetric.With(prometheus.Labels{"url": server.URL()}),
			)

			Expect(errorMetricValueAfter - errorMetricValueBefore).To(Equal(0.0))

			count := promtest.CollectAndCount(FetcherFetchesMetric)
			Expect(count).NotTo(Equal(0))

			petitionCount := promtest.CollectAndCount(SignaturesTotalMetric)
			Expect(petitionCount).NotTo(Equal(4))
		})
	})

	Context("when the petitions is not working", func() {
		BeforeEach(func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/petitions.json", "state=open"),
					ghttp.RespondWith(http.StatusNotFound, `{
						"message": "404 democracy not found"
					}`),
				),
			)
		})

		It("should expose petitions as metrics", func() {
			errorMetricValueBefore := promtest.ToFloat64(
				FetcherErrorsTotalMetric.With(prometheus.Labels{"url": server.URL()}),
			)

			fetcher.Start()

			Eventually(server.ReceivedRequests).Should(HaveLen(1))

			fetcher.Stop()
			fetcher.Wait()

			errorMetricValueAfter := promtest.ToFloat64(
				FetcherErrorsTotalMetric.With(prometheus.Labels{"url": server.URL()}),
			)

			Expect(errorMetricValueAfter - errorMetricValueBefore).To(Equal(1.0))
		})
	})
})
