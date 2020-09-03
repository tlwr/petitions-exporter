package main_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

func TestPetitionsFetcher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PetitonsExporter")
}

var _ = Describe("Exporter", func() {
	var (
		err     error
		binPath string
		session *gexec.Session

		server *ghttp.Server
	)

	BeforeSuite(func() {
		binPath, err = gexec.Build("github.com/tlwr/petitions-exporter")
		Expect(err).ShouldNot(HaveOccurred())
	})

	AfterSuite(func() {
		gexec.CleanupBuildArtifacts()
	})

	BeforeEach(func() {
		server = ghttp.NewServer()

		command := exec.Command(binPath)

		command.Env = append(
			os.Environ(),
			fmt.Sprintf("PETITIONS_URL=%s", server.URL()),
		)

		session, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).ShouldNot(HaveOccurred())

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

		Eventually(func() int {
			resp, err := http.Get("http://localhost:8080/health")
			if err != nil {
				return 0
			}
			return resp.StatusCode
		}).Should(Equal(200))
	})

	AfterEach(func() {
		server.Close()
		session.Terminate().Wait()
	})

	It("should export metrics", func() {
		resp, err := http.Get("http://localhost:8080/metrics")
		Expect(err).NotTo(HaveOccurred())

		body, err := ioutil.ReadAll(resp.Body)
		Expect(err).NotTo(HaveOccurred())
		resp.Body.Close()

		Expect(string(body)).To(MatchRegexp(fmt.Sprintf(
			`petitions_signatures{action="change a thing",id="3",opened_at="[^"]*",petition_url="%s/petitions/3",url="%s"} 789`,
			server.URL(),
			server.URL(),
		)))
	})
})
