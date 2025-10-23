package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
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
	switch method {
	case "POST", "PUT":
		body, err = json.Marshal(req)
		if err != nil {
			fmt.Println("Marshal error:", err)
			return ""
		}
	case "PATCH":
		body = []byte(`{}`)
	}

	var reqObj *http.Request
	if len(body) > 0 {
		reqObj, err = http.NewRequest(method, reqURL, bytes.NewBuffer(body))
	} else {
		reqObj, err = http.NewRequest(method, reqURL, nil)
	}
	if err != nil {
		fmt.Println("Request error:", err)
		return ""
	}
	reqObj.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(reqObj)
	if err != nil {
		fmt.Printf("[%s] Error: %v\n", method, err)
		return ""
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("[%s] %s => %s\n", method, reqURL, string(result))

	// Nếu là POST, lấy id dau
	if method == "POST" {
		var apiResp map[string]interface{}
		if err := json.Unmarshal(result, &apiResp); err == nil {
			if arr, ok := apiResp["data"].([]interface{}); ok && len(arr) > 0 {
				if first, ok := arr[0].(map[string]interface{}); ok {
					if realID, ok := first["id"].(string); ok {
						return realID
					}
				}
			}
		}
	}
	return id
}

func doScenario(client *http.Client, clientID string, measurement string) {
	req := QoSCreateRequest{ClientID: clientID, MeasurementType: measurement, Duration: 50}
	realID := sendRequest(client, "POST", req, clientID)

	if realID == "" {
		fmt.Println("POST failed")
		return
	}

	for i := 0; i < 5; i++ {
		method := randomMethod()
		sendRequest(client, method, req, realID)

		if method == "DELETE" {
			realID = sendRequest(client, "POST", req, clientID)
		}
	}
}

func shortConnection(clientIndex int) {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
		DialContext:       (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
	}
	client := &http.Client{Transport: transport}
	doScenario(client, fmt.Sprintf("client_short_%d", clientIndex), "real-time")
}

func pooledConnection(clientIndex int) {
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
	}
	client := &http.Client{Transport: transport}
	doScenario(client, fmt.Sprintf("client_pool_%d", clientIndex), "stress-test")
}

func main() {
	if err := CreateCSVHeader("qos_training_data.csv"); err != nil {
		fmt.Printf("Error creating CSV header: %v\n", err)
	}

	for i := 0; i < 10; i++ {
		shortConnection(i)
		pooledConnection(i)
		fmt.Printf("Round %d completed\n", i+1)
		time.Sleep(2 * time.Second)
	}
}
