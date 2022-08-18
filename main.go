package main

import (
	"context"
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
		nameEnvKey          = "VECRO_NAME"
		subsystemEnvKey     = "VECRO_SUBSYSTEM"
		listenAddressEnvKey = "VECRO_LISTEN_ADDRESS"
		callEnvKey          = "VECRO_CALLS"
		callSeparator       = " "
	)

	const (
		workloadCPUEnvKey         = "VECRO_WORKLOAD_CPU"
		workloadIOEnvKey          = "VECRO_WORKLOAD_IO"
		workloadDelayTimeEnvKey   = "VECRO_WORKLOAD_DELAY_TIME"
		workloadDelayJitterEnvKey = "VECRO_WORKLOAD_DELAY_JITTER"
		workloadNetEnvKey         = "VECRO_WORKLOAD_NET"
		workloadMemoryEnvKey      = "VECRO_WORKLOAD_MEMORY"
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
		memLoad     int
	)
	delayTime, _ = getEnvInt(workloadDelayTimeEnvKey, 0)
	delayJitter, _ = getEnvInt(workloadDelayJitterEnvKey, delayTime/10)
	cpuLoad, _ = getEnvInt(workloadCPUEnvKey, 0)
	ioLoad, _ = getEnvInt(workloadIOEnvKey, 0)
	netLoad, _ = getEnvInt(workloadNetEnvKey, 0)
	memLoad, _ = getEnvInt(workloadMemoryEnvKey, 0)

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
		Namespace: "vecro_base",
		Subsystem: subsystem,
		Name:      "request_count",
		Help:      "Number of requests received.",
		ConstLabels: map[string]string{
			"vecrosim_service_name": name,
		},
	}, nil)
	latencyCounter := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "vecro_base",
		Subsystem: subsystem,
		Name:      "latency_counter",
		Help:      "Processing time taken of requests in seconds, as counter.",
		ConstLabels: map[string]string{
			"vecrosim_service_name": name,
		},
	}, nil)
	latencyHistogram := kitprometheus.NewHistogramFrom(stdprometheus.HistogramOpts{
		Namespace: "vecro_base",
		Subsystem: subsystem,
		Name:      "latency_histogram",
		Help:      "Processing time taken of requests in seconds, as histogram.",
		// TODO: determine appropriate buckets
		Buckets: []float64{.0002, .001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 15, 25},
		ConstLabels: map[string]string{
			"vecrosim_service_name": name,
		},
	}, nil)
	throughput := kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
		Namespace: "vecro_base",
		Subsystem: subsystem,
		Name:      "throughput",
		Help:      "Size of data transmitted in bytes.",
		ConstLabels: map[string]string{
			"vecrosim_service_name": name,
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

	// Simulate memory allocation
	memBlock := allocMemory(memLoad)
	_ = memBlock

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
	svc = instrumentingMiddleware(requestCount, latencyCounter, latencyHistogram, logger)(svc)

	baseHandler := httptransport.NewServer(
		makeBaseEndPoint(svc),
		decodeBaseRequest,
		encodeResponse,
		// Request throughput instrumentation
		httptransport.ServerFinalizer(func(ctx context.Context, code int, r *http.Request){
			responseSize := ctx.Value(httptransport.ContextKeyResponseSize).(int64)
			logger.Log("reponse_size", responseSize)
			throughput.Add(float64(responseSize))
		}),
	)

	http.Handle("/", baseHandler)
	http.Handle("/metrics", promhttp.Handler())
	logger.Log("msg", "HTTP", "addr", listenAddress)
	logger.Log("err", http.ListenAndServe(listenAddress, nil))
}
