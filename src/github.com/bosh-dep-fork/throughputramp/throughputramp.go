package main

import (
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	numRequests      = flag.Int("n", 1000, "number of requests to send")
	interval         = flag.Int("i", 1, "interval in seconds to average throughput")
	threadRateLimit  = flag.Int("q", 0, "thread rate limit")
	lowerConcurrency = flag.Int("lower-concurrency", 1, "Starting concurrency value")
	upperConcurrency = flag.Int("upper-concurrency", 30, "Ending concurrency value")
	concurrencyStep  = flag.Int("concurrency-step", 1, "Concurrency increase per run")
	localCSV = flag.String("local-csv", "", "Stores csv locally to a specified directory when the flag is set")
	heyPath  = flag.String("hey-path", "hey", "Path to hey test hey")
)

func main() {
	flag.Parse()
	if flag.NArg() < 1 {
		usageAndExit()
	}

	router := flag.Args()[0]

	runBenchmark(router,
		*heyPath,
		*numRequests,
		*lowerConcurrency,
		*upperConcurrency,
		*concurrencyStep,
		*threadRateLimit)

}

func writeFile(path string, data []byte) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Creating csv file error: %s\n", err)
		os.Exit(1)
	}
	_, err = f.Write(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Writing csv data to a file error: %s\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "csv stored locally in file %s\n", path)
}

func runBenchmark(router,
	heyPath string,
	numRequests,
	lowerConcurrency,
	upperConcurrency,
	concurrencyStep,
	threshold int) {

	benchmarkData := new(bytes.Buffer)
	for i := lowerConcurrency; i <= upperConcurrency; i += concurrencyStep {
		heyData, benchmarkErr := run(router, heyPath, numRequests, i, threshold)

		if benchmarkErr != nil {
			fmt.Fprintf(os.Stderr, "%s\n", benchmarkErr)
			os.Exit(1)
		}

		_, writeErr := benchmarkData.Write(heyData)
		if writeErr != nil {
			fmt.Fprintf(os.Stderr, "Buffer error: %s\n", writeErr)
			os.Exit(1)
		}
	}

	if *localCSV != "" {
		perfResult := filepath.Join(*localCSV, "perfResults.csv")
		writeFile(perfResult, benchmarkData.Bytes())

	}
}

func run(router, heyPath string, numRequests, concurrentRequests, rateLimit int) ([]byte, error) {
	fmt.Fprintf(os.Stdout, "Running benchmark with %d requests, %d concurrency, and %d rate limit\n", numRequests, concurrentRequests, rateLimit)
	args := []string{
		"-n", strconv.Itoa(numRequests),
		"-c", strconv.Itoa(concurrentRequests),
		"-q", strconv.Itoa(rateLimit),
		"-o", "csv",
		"-t", "0",
		router,
	}

	heyData, err := exec.Command(heyPath, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("hey error: %s\nData:\n%s", err, string(heyData))
	}

    if strings.Contains(strings.ToUpper(string(heyData)), "ERROR DISTRIBUTION") {
	    return nil, fmt.Errorf("hey error: %s\n", string(heyData))
	}

	return selectCSVColumns(string(heyData)), nil
}

func selectCSVColumns(heyData string) []byte {
	const (
		startTime    = 0
		responseTime = 1
	)
	r := csv.NewReader(strings.NewReader(heyData))

	records, err := r.ReadAll()
	if err != nil {
		fmt.Errorf("reading csv records %s", err)
	}
	if len(records) == 0 {
		return nil
	}
	var b bytes.Buffer
	b.Write([]byte("start-time,response-time\n"))
	for i := 1; i < len(records); i++ {
		_, err = b.Write([]byte(fmt.Sprintf("%s,%s\n", records[i][startTime], records[i][responseTime])))
		if err != nil {
			fmt.Errorf("writing csv records %s", err)
		}
	}
	return b.Bytes()
}

func usageAndExit() {
	flag.Usage()
	fmt.Fprintf(os.Stderr, "\n")
	os.Exit(1)
}
