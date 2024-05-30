package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/endpoint"
)

func makeBaseEndPoint(svc BaseService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		payload, err := svc.Execute()
		return baseResponse{Payload: payload}, err
	}
}

func decodeBaseRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeBaseResponse(_ context.Context, r *http.Response) (interface{}, error) {
	return nil, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	return json.NewEncoder(w).Encode(response)
}

func encodeRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(request)
	if err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

type baseResponse struct {
	Payload string `json:"payload"`
}

type setDelayRequest struct {
	DelayTime int `json:"delay"`
	DelayJitter int `json:"jitter"`
}

func decodeSetDelayRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var request setDelayRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

func makeSetDelayEndPoint(svc BaseService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(setDelayRequest)
		payload, err := svc.SetDelay(req.DelayTime, req.DelayJitter)
		return baseResponse{Payload: payload}, err
	}
}
