package main

import (
	"context"
	"github.com/go-kit/kit/endpoint"
	"strings"
	"time"
	"fmt"
)

type BaseService interface {
	Execute() (string, error)
	SetDelay(int, int) (string, error)
}

type baseService struct {
	calls         []endpoint.Endpoint // Downstream endpoints to be called on
	isSynchronous bool                // Whether to call services synchronously or asynchronously.  TODO: NOT IMPLEMENTED YET!

	delayTime   int
	delayJitter int
	cpuLoad     int
	ioLoad      int
	netLoad     int
}

const executionTimeout = 120*time.Second // TODO: make service timeout configurable

func (svc baseService) Execute() (string, error) {
	// Establish the connection
	ctx, cancel := context.WithTimeout(context.Background(), executionTimeout)
	defer cancel()

	// Simulate Stress
	// TODO: Re-determine timings of stress
	stress(svc.delayTime, svc.delayJitter, svc.cpuLoad, svc.ioLoad)

	// Call downstream services
	for _, ep := range svc.calls {
		// TODO: Pass parameters and parse responses
		// TODO: Support async calling mode
		_, err := ep(ctx, nil)
		if err != nil {
			return "", err
		}
	}

	// Simulate response payload
	var payload string
	if svc.netLoad > 0 {
		payload = strings.Repeat("0", svc.netLoad/2)
	}

	// Return result
	return payload, nil
}


func (svc* baseService) SetDelay(delayTime int, delayJitter int) (string, error) {
	// Establish the connection
	_, cancel := context.WithTimeout(context.Background(), executionTimeout)
	defer cancel()

	// Overwrite old delay & jitter
	svc.delayTime = delayTime
	svc.delayJitter = delayJitter


	// Format response payload
	payload := fmt.Sprintf("delayTime and delayJitter set to %d and %d", delayTime, delayJitter)

	// Return result
	return payload, nil
}

type ServiceMiddleware func(BaseService) BaseService
