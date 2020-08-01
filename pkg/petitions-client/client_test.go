package client_test

import (
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/sirupsen/logrus"

	. "github.com/tlwr/petitions-exporter/pkg/petitions-client"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PetitionsClient")
}

var _ = Describe("Client", func() {
	var (
		server *ghttp.Server

		logger  *logrus.Logger
		baseURL string

		client Client
	)

	BeforeEach(func() {
		server = ghttp.NewServer()

		logger = logrus.New()
		logger.SetOutput(GinkgoWriter)
		logger.SetFormatter(&logrus.JSONFormatter{})

		baseURL = server.URL()

		client = New(baseURL, logger)
	})

	AfterEach(func() {
		server.Close()
	})

	It("should return some petitions", func() {
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

		petitions, err := client.List()

		Expect(err).NotTo(HaveOccurred())
		Expect(petitions).To(HaveLen(4))

		Expect(petitions[0].ID()).To(BeNumerically("~", 1))
		Expect(petitions[3].ID()).To(BeNumerically("~", 4))
		Expect(petitions[0].OpenedAt().Year()).To(Equal(2020))
		Expect(petitions[3].OpenedAt().Year()).To(Equal(2020))
		Expect(petitions[0].OpenedAt().Month()).To(BeNumerically("~", 7))
		Expect(petitions[3].OpenedAt().Month()).To(BeNumerically("~", 7))
		Expect(petitions[0].OpenedAt().Day()).To(Equal(2))
		Expect(petitions[3].OpenedAt().Day()).To(Equal(5))
		Expect(petitions[0].Action()).To(Equal("start a thing"))
		Expect(petitions[3].Action()).To(Equal("return a thing"))
	})

	It("should handle an error", func() {
		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("GET", "/petitions.json", "state=open"),
				ghttp.RespondWith(http.StatusNotFound, `{
					"message": "404 democracy not found"
				}`),
			),
		)

		petitions, err := client.List()

		Expect(err).To(HaveOccurred())
		Expect(petitions).To(HaveLen(0))
	})
})
