package main

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gdharley/flowable-external-client-golang/flowable"
)

// External worker callback - This is the worker handler function
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
		if err := json.Unmarshal([]byte(body), &data); err != nil {
			// Unmarshal failed — mark handler as failed and set error code
			res.ErrorCode = err.Error()
			res.Status = flowable.HandlerFail
		}
	}

	// Add a dummy variable for testing/visibility
	res.Variables = append(res.Variables, flowable.HandlerVariable{Name: "dummy", Type: "string", Value: "a simple string"})

	return res.Status, res
}

func main() {
	url := "http://localhost:8090/flowable-work"
	interval := 5 * time.Second

	// Configure package-level defaults for auth/headers
	flowable.SetAuth("admin", "test")
	// (optional) override headers if needed
	// flowable.SetDefaultHeader("X-My-Header", "value")

	// Start polling in a goroutine to avoid blocking, provide acquire params
	acquireParams := flowable.AcquireRequest{
		Topic:           "order",
		LockDuration:    "PT10M",
		NumberOfTasks:   1,
		NumberOfRetries: 10,
		WorkerId:        "orderWorker1",
		ScopeType:       "cmmn",
	}
	go flowable.Subscribe(url, interval, external_worker, acquireParams)
	// To perform a single GET using List_jobs:
	// status, body, err := flowable.List_jobs(url)
	// fmt.Println(status, body, err)

	// Keep the main function running indefinitely
	select {}
}
