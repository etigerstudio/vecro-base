package main

import (
	"github.com/go-kit/kit/log"
	"time"

	"github.com/go-kit/kit/metrics"
)

func instrumentingMiddleware(
	requestCount metrics.Counter,
	latencyCounter metrics.Counter,
	latencyHistogram metrics.Histogram,
	logger log.Logger,
) ServiceMiddleware {
	return func(next BaseService) BaseService {
		return instrmw{
			requestCount,
			latencyCounter,
			latencyHistogram,
			logger,
			next,
		}
	}
}

type instrmw struct {
	requestCount     metrics.Counter
	latencyCounter   metrics.Counter
	latencyHistogram metrics.Histogram
	logger           log.Logger
	BaseService
}

func (mw instrmw) Execute() (string, error) {
	defer func(begin time.Time) {
		mw.requestCount.Add(1)
		mw.latencyCounter.Add(time.Since(begin).Seconds())
		mw.latencyHistogram.Observe(time.Since(begin).Seconds())
		mw.logger.Log("request_latency:", time.Since(begin))
	}(time.Now())

	return mw.BaseService.Execute()
}
