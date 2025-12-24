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
		} else {
			// Add a dummy variable for testing/visibility
			res.Variables = append(res.Variables, flowable.HandlerVariable{Name: "dummy", Type: "string", Value: "a simple string"})
			res.Status = flowable.HandlerSuccess
		}
	}

	return res.Status, res
}

func main() {
	url := "http://localhost:8090"
	interval := 10 * time.Second

	// Configure package-level defaults for auth/headers
	flowable.SetAuth("admin", "test")
	// (optional) override headers if needed
	// flowable.SetDefaultHeader("X-My-Header", "value")

	// Start polling in a goroutine to avoid blocking, provide acquire params
	acquireParams := flowable.AcquireRequest{
		Topic:           "testing",
		LockDuration:    "PT10M",
		NumberOfTasks:   1,
		NumberOfRetries: 5,
		WorkerId:        "worker1",
		ScopeType:       "bpmn",
	}
	go flowable.Subscribe(url, interval, external_worker, acquireParams)

	// Keep the main function running indefinitely
	select {}
}
