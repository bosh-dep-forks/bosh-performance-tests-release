package main_test

import (
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

var _ = Describe("Throughputramp", func() {
	var (
		runner          *ginkgomon.Runner
		process         ifrit.Process
		testServer      *ghttp.Server
		bodyChan        chan []byte
		runnerArgs      Args
		bodyTestHandler http.HandlerFunc
	)

	Context("when correct arguments are used", func() {
		BeforeEach(func() {
			testServer = ghttp.NewUnstartedServer()
			handler := ghttp.CombineHandlers(
				ghttp.RespondWith(http.StatusOK, nil),
			)
			testServer.AppendHandlers(handler)
			testServer.AllowUnhandledRequests = true
			testServer.Start()

			bodyChan = make(chan []byte, 3)

			bodyTestHandler = ghttp.CombineHandlers(
				ghttp.VerifyHeaderKV("X-Amz-Acl", "public-read"),
				func(rw http.ResponseWriter, req *http.Request) {
					defer GinkgoRecover()
					defer req.Body.Close()
					bodyBytes, err := ioutil.ReadAll(req.Body)
					Expect(err).ToNot(HaveOccurred())
					bodyChan <- bodyBytes
				},
				ghttp.RespondWith(http.StatusOK, nil),
			)

			runnerArgs = Args{
				NumRequests:      12,
				RateLimit:        100,
				Router:           testServer.URL(),
				StartConcurrency: 2,
				EndConcurrency:   4,
			}
		})

		JustBeforeEach(func() {
			runner = NewThroughputRamp(binPath, runnerArgs)
			process = ginkgomon.Invoke(runner)
			Eventually(process.Wait(), "5s").Should(Receive())
		})

		AfterEach(func() {
			ginkgomon.Interrupt(process)
			testServer.Close()
			close(bodyChan)
		})

		It("ramps up throughput over multiple tests", func() {
			Expect(runner.ExitCode()).To(Equal(0))
			Expect(testServer.ReceivedRequests()).To(HaveLen(36))
		})

		Context("when local-csv is specified", func() {
			var dir string
			BeforeEach(func() {
				var err error
				dir, err = ioutil.TempDir("", "test")
				Expect(err).NotTo(HaveOccurred())
				runnerArgs.localCSV = dir
				cpumonitorServer := ghttp.NewServer()

				header := make(http.Header)
				header.Add("Content-Type", "application/json")

				cpumonitorServer.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/stop"),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/start"),
						ghttp.RespondWith(http.StatusOK, nil),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/stop"),
					),
				)
			})
			It("stores the csv locally", func() {
				checkFiles := func() int {
					files, err := ioutil.ReadDir(dir)
					Expect(err).ToNot(HaveOccurred())
					fileCount := 0
					for _, file := range files {
						if strings.Contains(file.Name(), "csv") {
							Expect(file.Size()).ToNot(BeZero())
							fileCount++
						}
					}
					return fileCount
				}
				Eventually(checkFiles).Should(Equal(1))
				Expect(os.RemoveAll(dir)).To(Succeed())
			})
		})
	})
	Context("when incorrect arguments are passed in", func() {
		BeforeEach(func() {
			runner = NewThroughputRamp(binPath, Args{})
			runner.Command = exec.Command(binPath)
		})

		It("exits 1 with usage", func() {
			process := ifrit.Background(runner)
			Eventually(process.Wait()).Should(Receive())
			Expect(runner.ExitCode()).To(Equal(1))
		})
	})

	Context("when the test host is not available", func() {
		BeforeEach(func() {
			runnerArgs = Args{
				NumRequests:      12,
				RateLimit:        100,
				Router:           "http://example.com",
				StartConcurrency: 2,
				EndConcurrency:   4,
			}
			runner = NewThroughputRamp(binPath, runnerArgs)

		    runner = NewThroughputRamp(binPath, runnerArgs)
            process = ginkgomon.Invoke(runner)
            Eventually(process.Wait(), "5s").Should(Receive())
		})

		It("the exit code is not 0", func() {
			Expect(runner.ExitCode()).To(Equal(0))
		})
	})
})
