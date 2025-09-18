package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Short connection
func shortConnection() {
	req := QoSCreateRequest{ClientID: "client1", MeasurementType: "real-time", Duration: 10}
	body, _ := json.Marshal(req)

	resp, err := http.Post("http://localhost:8080/qos", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	result, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Short connection response:", string(result))
}

// Connection pool
func pooledConnection() {
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
	}
	client := &http.Client{Transport: transport}

	for i := 0; i < 10; i++ {
		req := QoSCreateRequest{ClientID: fmt.Sprintf("client_pool_%d", i), MeasurementType: "stress-test", Duration: 5}
		body, _ := json.Marshal(req)

		resp, err := client.Post("http://localhost:8080/qos", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		result, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Println("Pooled connection response:", string(result))
	}
}

func main() {
	fmt.Println("Short connection\n")
	shortConnection()

	fmt.Println("Pooled connection\n")
	pooledConnection()
}
