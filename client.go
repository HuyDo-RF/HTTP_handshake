package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Short connection
func shortConnection() {
	transport := &http.Transport{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		DisableKeepAlives: true,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
	}
	client := &http.Client{Transport: transport}

	req := QoSCreateRequest{ClientID: "client1", MeasurementType: "real-time", Duration: 10}
	body, _ := json.Marshal(req)

	resp, err := client.Post("https://127.0.0.1:8080/qos", "application/json", bytes.NewBuffer(body))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()
	result, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("Short connection response:", string(result))
}

// Connection pool (tái sử dụng kết nối)
func pooledConnection() {
	transport := &http.Transport{
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
		DisableKeepAlives:   false,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
	}

	client := &http.Client{Transport: transport}

	for i := 0; i < 10; i++ {
		req := QoSCreateRequest{
			ClientID:        fmt.Sprintf("client_pool_%d", i),
			MeasurementType: "stress-test",
			Duration:        5,
		}
		body, _ := json.Marshal(req)

		resp, err := client.Post("https://127.0.0.1:8080/qos", "application/json", bytes.NewBuffer(body))
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}
		result, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		fmt.Printf("Pooled connection response %d: %s\n", i, string(result))
	}
}

func main() {
	fmt.Println("Short connection\n")
	shortConnection()

	fmt.Println("\nPooled connection\n")
	pooledConnection()
}
