package main_test

import (
	"os/exec"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit/ginkgomon"

	"testing"
)

var (
	binPath string
	heyPath string
)

func NewThroughputRamp(throughputRampPath string, args Args) *ginkgomon.Runner {
	return ginkgomon.New(ginkgomon.Config{
		Name:    "throughputramp",
		Command: exec.Command(throughputRampPath, args.ArgSlice()...),
	})
}

type Args struct {
	NumRequests      int
	RateLimit        int
	StartConcurrency int
	EndConcurrency   int
	Router           string
	localCSV         string
}

func (args Args) ArgSlice() []string {
	argSlice := []string{
		"-n", strconv.Itoa(args.NumRequests),
		"-q", strconv.Itoa(args.RateLimit),
		"-lower-concurrency", strconv.Itoa(args.StartConcurrency),
		"-upper-concurrency", strconv.Itoa(args.EndConcurrency),
		"-local-csv", args.localCSV,
		"-hey-path", heyPath,
	}

	argSlice = append(argSlice, args.Router)
	return argSlice
}

func TestThroughputramp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Throughputramp Suite")
}

var _ = BeforeSuite(func() {
	var err error
	var heyPathError error
	heyPath, heyPathError = gexec.Build("../../rakyll/hey", "-race")
	Expect(heyPathError).NotTo(HaveOccurred())
	binPath, err = gexec.Build("github.com/bosh-dep-fork/throughputramp", "-race")
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})
