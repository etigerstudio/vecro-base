package main

import (
	"github.com/go-kit/kit/endpoint"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	httptransport "github.com/go-kit/kit/transport/http"
)

func main() {
	// -------------------
	// Declare constants
	// -------------------
	const (
		nameEnvKey          = "BEN_NAME"
		subsystemEnvKey     = "BEN_SUBSYSTEM"
		listenAddressEnvKey = "BEN_LISTEN_ADDRESS"
		callEnvKey          = "BEN_CALLS"
		callSeparator       = " "
	)

	const (
		workloadCPUEnvKey         = "BEN_WORKLOAD_CPU"
		workloadIOEnvKey          = "BEN_WORKLOAD_IO"
		workloadDelayTimeEnvKey   = "BEN_WORKLOAD_DELAY_TIME"
		workloadDelayJitterEnvKey = "BEN_WORKLOAD_DELAY_JITTER"
		workloadNetEnvKey         = "BEN_WORKLOAD_NET"
	)

	// -------------------
	// Init logging
	// -------------------
	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "caller", log.DefaultCaller)

	// -------------------
	// Parse Environment variables
	// -------------------
	var (
		delayTime   int
		delayJitter int
		cpuLoad     int
		ioLoad      int
		netLoad     int
	)
	delayTime, _ = getEnvInt(workloadDelayTimeEnvKey, 0)
	delayJitter, _ = getEnvInt(workloadDelayJitterEnvKey, delayTime/10)
	cpuLoad, _ = getEnvInt(workloadCPUEnvKey, 0)
	ioLoad, _ = getEnvInt(workloadIOEnvKey, 0)
	netLoad, _ = getEnvInt(workloadNetEnvKey, 0)

	logger.Log("delay time", delayTime)
	logger.Log("delay jitter", delayJitter)
	logger.Log("cpu load", cpuLoad)
	logger.Log("io load", ioLoad)
	logger.Log("net load", netLoad)

	listenAddress, _ := getEnvString(listenAddressEnvKey, ":8080")
	logger.Log("listen_address", listenAddress)

	subsystem, _ := getEnvString(subsystemEnvKey, "subsystem")
	name, _ := getEnvString(nameEnvKey, "name")
	logger.Log("name", name, "subsystem", subsystem)

	// -------------------
	// Init Prometheus counter & histogram
	// -------------------
	requestCount := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "ben_base",
		Subsystem: subsystem,
		Name:      "request_count",
		Help:      "Number of requests received.",
		ConstLabels: map[string]string{
			"bensim_service_name": name,
		},
	}, nil)
	requestLatency := kitprometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Namespace: "ben_base",
		Subsystem: subsystem,
		Name:      "request_latency",
		Help:      "Total duration of requests in microseconds.",
		// TODO: determine appropriate buckets
		Buckets: []float64{.0002, .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 25},
		ConstLabels: map[string]string{
			"bensim_service_name": name,
		},
	}, nil)

	// -------------------
	// Init call endpoints
	// -------------------

	// Create call endpoint list from the environment variable
	var calls []endpoint.Endpoint
	callList, exists := getEnvString(callEnvKey, "")
	if exists {
		logger.Log("calls", callList)

		for _, callStr := range strings.Split(callList, callSeparator) {
			callURL, err := url.Parse(callStr)
			if err != nil {
				panic(err)
			}
			callEndpoint := httptransport.NewClient(
				"GET",
				callURL,
				encodeRequest,
				decodeBaseResponse,
			).Endpoint()
			calls = append(calls, callEndpoint)
		}
	} else {
		logger.Log("calls", "[empty call list]")
	}

	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	// -------------------
	// Create & run service
	// -------------------
	var svc BaseService
	svc = baseService{
		calls:       calls,
		delayTime:   delayTime,
		delayJitter: delayJitter,
		cpuLoad:     cpuLoad,
		ioLoad:      ioLoad,
		netLoad:     netLoad,
	}
	svc = loggingMiddleware(logger)(svc)
	svc = instrumentingMiddleware(requestCount, requestLatency, logger)(svc)

	baseHandler := httptransport.NewServer(
		makeBaseEndPoint(svc),
		decodeBaseRequest,
		encodeResponse,
	)

	http.Handle("/", baseHandler)
	http.Handle("/metrics", promhttp.Handler())
	logger.Log("msg", "HTTP", "addr", listenAddress)
	logger.Log("err", http.ListenAndServe(listenAddress, nil))
}
