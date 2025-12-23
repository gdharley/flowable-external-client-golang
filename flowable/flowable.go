package flowable

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
)

const job_api = "/external-job-api"

// HandlerStatus indicates how the handler processed a job.
type HandlerStatus string

const (
	HandlerSuccess       HandlerStatus = "success"
	HandlerFail          HandlerStatus = "fail"
	HandlerBPMNError     HandlerStatus = "bpmnError"
	HandlerCMMNTerminate HandlerStatus = "cmmnTerminate"
)

// HandlerVariable represents a single variable in the handler result.
type HandlerVariable struct {
	Name  string      `json:"name"`
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// HandlerResult is the structured response returned by the handler.
type HandlerResult struct {
	Status    HandlerStatus     `json:"status"`
	WorkerId  string            `json:"workerId,omitempty"`
	Variables []HandlerVariable `json:"variables"`
	ErrorCode string            `json:"errorCode,omitempty"`
}

// Callback function type. The handler returns a HandlerStatus and an optional structured result.
type ResponseHandler func(status int, body string) (HandlerStatus, *HandlerResult)

// AcquireRequest represents the body sent to the acquire endpoint.
type AcquireRequest struct {
	Topic           string `json:"topic"`
	LockDuration    string `json:"lockDuration"`
	NumberOfTasks   int    `json:"numberOfTasks"`
	NumberOfRetries int    `json:"numberOfRetries"`
	WorkerId        string `json:"workerId"`
	ScopeType       string `json:"scopeType"`
}

// restGet performs a GET request to the provided full URL and returns status, body bytes, and error.
func restGet(fullURL string) (status int, body []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return -1, nil, err
	}
	req.Header.Add("Accept", `application/json`)
	req.Header.Add("Content-Type", `application/json`)
	req.SetBasicAuth("admin", "test")

	resp, err := client.Do(req)
	if err != nil {
		return -1, nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, bodyBytes, nil
}

// restPost performs a POST request to the provided full URL with the given JSON payload.
func restPost(fullURL string, payload []byte) (status int, body []byte, err error) {
	client := &http.Client{}
	req, err := http.NewRequest("POST", fullURL, bytes.NewReader(payload))
	if err != nil {
		return -1, nil, err
	}
	req.Header.Add("Accept", `application/json`)
	req.Header.Add("Content-Type", `application/json`)
	req.SetBasicAuth("admin", "test")

	resp, err := client.Do(req)
	if err != nil {
		return -1, nil, err
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, bodyBytes, nil
}

// Acquire_jobs performs a POST to the acquire jobs endpoint (/acquire/jobs) with a JSON body.
func Acquire_jobs(url string, reqBody AcquireRequest) (jobs []interface{}, body string, status int, err error) {
	full := url + job_api + "/acquire/jobs"
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", -1, err
	}
	status, bodyBytes, err := restPost(full, payload)
	if err != nil {
		return nil, "", status, err
	}

	var parsed []interface{}
	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		// If response isn't a JSON array, return an error
		return nil, string(bodyBytes), status, err
	}
	return parsed, string(bodyBytes), status, nil
}

// List_jobs performs a single GET to the jobs endpoint and returns the status and raw response body.
func List_jobs(url string) (status int, body string, err error) {
	full := url + job_api + "/jobs"
	status, bodyBytes, err := restGet(full)
	if err != nil {
		return -1, "", err
	}
	return status, string(bodyBytes), nil
}

// Subscribe polls the given URL at intervals and invokes the handler when jobs are available.
// acquireReq must be provided by the caller with the desired acquire parameters.
func Subscribe(url string, interval time.Duration, handler ResponseHandler, acquireReq AcquireRequest) {
	for {
		jobs, _, status, err := Acquire_jobs(url, acquireReq)
		if err != nil {
			// If acquire failed (including parse errors), treat as status 500 and pass the raw body if available
			resStatus, resObj := handler(500, "")
			log.Printf("handler returned on acquire error: %s, result=%v", resStatus, resObj)
			time.Sleep(interval)
			continue
		}
		if len(jobs) == 0 {
			// No jobs, wait and poll again
			time.Sleep(interval)
			continue
		}
		// Jobs found, invoke handler for each job individually
		for _, job := range jobs {
			jobBytes, err := json.Marshal(job)
			if err != nil {
				// If we can't serialize an individual job, treat as processing/parsing failure => status 500
				resStatus, resObj := handler(500, "")
				log.Printf("handler returned serializing job: %s, result=%v", resStatus, resObj)
				continue
			}
			resStatus, resObj := handler(status, string(jobBytes))
			// If the handler returned a structured result, log it (or handle it here)
			if resObj != nil {
				if jb, err := json.Marshal(resObj); err == nil {
					log.Printf("handler returned for job: status=%s, result=%s", resStatus, string(jb))
				} else {
					log.Printf("handler returned for job: status=%s, could not marshal result: %v", resStatus, err)
				}
			} else {
				log.Printf("handler returned for job: status=%s, result=nil", resStatus)
			}
		}
		time.Sleep(interval)
	}
}
