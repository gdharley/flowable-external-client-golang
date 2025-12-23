package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gdharley/flowable-external-client-golang/flowable"
)

// External worker callback - simply unmarshal body into JSON (or keep raw string) and return the structured result
func external_worker(status int, body string) (flowable.HandlerStatus, *flowable.HandlerResult) {

	// Initialize the response object
	res := &flowable.HandlerResult{
		Status:    flowable.HandlerSuccess,
		WorkerId:  "",
		Variables: []flowable.HandlerVariable{},
		ErrorCode: "",
	}

	// Either Acquire_jobs or the job itself could not be parsed
	if status >= 400 {
		res.ErrorCode = strconv.Itoa(status)
		res.Status = flowable.HandlerFail
	}

	if body != "" {
		var data interface{}
		if err := json.Unmarshal([]byte(body), &data); err == nil {
			res.Variables = append(res.Variables, flowable.HandlerVariable{Name: "body", Type: "json", Value: data})
		} else {
			res.Variables = append(res.Variables, flowable.HandlerVariable{Name: "body", Type: "string", Value: body})
		}
	}

	return res.Status, res
}

func main() {
	url := "http://localhost:8090/flowable-work"
	interval := 5 * time.Second

	// use externalized callback
	callback := external_worker

	// Start polling in a goroutine to avoid blocking, provide acquire params
	acquireParams := flowable.AcquireRequest{
		Topic:           "order",
		LockDuration:    "PT10M",
		NumberOfTasks:   1,
		NumberOfRetries: 10,
		WorkerId:        "orderWorker1",
		ScopeType:       "cmmn",
	}
	go flowable.Subscribe(url, interval, callback, acquireParams)
	// To perform a single GET using List_jobs:
	// status, body, err := flowable.List_jobs(url)
	// fmt.Println(status, body, err)

	// Keep the main function running indefinitely
	select {}
}
