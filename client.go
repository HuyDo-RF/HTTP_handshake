package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"time"
)

func randomMethod() string {
	methods := []string{"GET", "PUT", "PATCH", "DELETE"}
	rand.Seed(time.Now().UnixNano())
	return methods[rand.Intn(len(methods))]
}

func sendRequest(client *http.Client, method string, req QoSCreateRequest, id string) string {
	baseURL := "https://127.0.0.1:8080/qos"
	var reqURL string
	if method == "POST" {
		reqURL = baseURL
	} else {
		reqURL = baseURL + "/" + id
	}

	var body []byte
	var err error
	if method == "POST" || method == "PUT" || method == "PATCH" {
		body, err = json.Marshal(req)
		if err != nil {
			fmt.Println("Marshal error:", err)
			return ""
		}
	}

	var reqBody io.Reader
	if method == "POST" || method == "PUT" || method == "PATCH" {
		reqBody = bytes.NewBuffer(body)
	}

	request, err := http.NewRequest(method, reqURL, reqBody)
	if err != nil {
		fmt.Println("Request error:", err)
		return ""
	}
	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		fmt.Printf("[%s] Error: %v\n", method, err)
		return ""
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("[%s] %s => %s\n", method, reqURL, string(result))

	if method == "POST" {
		var apiResp map[string]interface{}
		if err := json.Unmarshal(result, &apiResp); err == nil {
			if dataMap, ok := apiResp["data"].(map[string]interface{}); ok {
				if realID, ok := dataMap["id"].(string); ok {
					return realID
				}
			}
		}
	}
	return id
}

func doScenario(client *http.Client, clientID string, measurement string) {
	req := QoSCreateRequest{ClientID: clientID, MeasurementType: measurement, Duration: 10}
	realID := sendRequest(client, "POST", req, clientID)

	if realID == "" {
		fmt.Println("POST failed")
		return
	}

	for i := 0; i < 4; i++ {
		method := randomMethod()
		sendRequest(client, method, req, realID)
		if method == "DELETE" {
			realID = sendRequest(client, "POST", req, clientID)
		}
	}
}

func shortConnection() {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
		DialContext:       (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
	}
	client := &http.Client{Transport: transport}
	doScenario(client, "client_short", "real-time")
}

func pooledConnection() {
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
	}
	client := &http.Client{Transport: transport}

	for i := 0; i < 3; i++ {
		id := fmt.Sprintf("client_pool_%d", i)
		doScenario(client, id, "stress-test")
	}
}

func main() {
	fmt.Println("Short connection test")
	shortConnection()
	fmt.Println("\nPooled connection test")
	pooledConnection()
}
