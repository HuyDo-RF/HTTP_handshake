package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	qosDB         = make(map[string]*database)
	lock          sync.Mutex
	lastTimestamp time.Time
)

func main() {
	r := gin.Default()

	//POST: Tich hop AI
	r.POST("/qos", func(c *gin.Context) {
		var req QoSCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, APIresponse{
				Success: false, Message: "Invalid request", Error: err.Error(),
			})
			return
		}

		lock.Lock()
		now := time.Now().UTC()
		var start time.Time
		// đảm bảo global monotonic
		if lastTimestamp.IsZero() || now.After(lastTimestamp.Add(1*time.Minute)) {
			start = now
		} else {
			start = lastTimestamp.Add(1 * time.Minute)
		}

		series := GenerateQoSSeriesFromStart(req.ClientID, req.MeasurementType, req.Duration, start)
		for _, d := range series {
			qosDB[d.ID] = d
		}
		lastTimestamp = start.Add(time.Duration(req.Duration-1) * time.Minute)

		if err := SaveQoSSeriesToCSV(series, "qos_training_data.csv"); err != nil {
			fmt.Printf("Error saving to CSV: %v\n", err)
		}
		lock.Unlock()

		//Gọi Python
		cmd := exec.Command("D:\\5G VDT\\HTTP_VHT\\QoS_HTTP2\\.venv\\Scripts\\python.exe", "predict_qos.py")
		inputJSON, _ := json.Marshal(series)

		cmd.Stdin = bytes.NewReader(inputJSON)
		cmd.Stderr = nil
		out, err := cmd.Output()

		fmt.Println("=== Python Output ===")
		fmt.Println(string(out))
		fmt.Println("=====================")

		if err != nil {
			fmt.Printf("Predict model error: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":         err.Error(),
				"python_output": string(out),
			})
			return
		}

		//Parse output JSON từ Python
		var result map[string]interface{}
		if err := json.Unmarshal(out, &result); err != nil {
			fmt.Printf("JSON parse error from Python output: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Cannot parse Python output",
				"raw":   string(out),
			})
			return
		}

		/*if err != nil {
			fmt.Printf("Predict model error: %v\n", err)
		} else {
			var pred map[string]interface{}
			if json.Unmarshal(out, &pred) == nil {
				c.JSON(http.StatusOK, gin.H{
					"success":    true,
					"message":    "QoS series generated & evaluated",
					"alerts":     pred["alerts"],
					"prediction": pred["prediction"],
				})
				return
			}
		} */

		c.JSON(http.StatusOK, APIresponse{
			Success: true, Message: "QoS series generated", Data: series,
		})
	})

	r.GET("/qos/:id", func(c *gin.Context) {
		id := c.Param("id")

		lock.Lock()
		data, exists := qosDB[id]
		lock.Unlock()

		if !exists {
			c.JSON(http.StatusNotFound, APIresponse{Success: false, Message: "Data not found"})
			return
		}
		c.JSON(http.StatusOK, APIresponse{Success: true, Data: data})
	})

	r.PUT("/qos/:id", func(c *gin.Context) {
		id := c.Param("id")
		var req QoSCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, APIresponse{Success: false, Message: "Invalid request", Error: err.Error()})
			return
		}

		series := GenerateQoSSeriesFromStart(req.ClientID, req.MeasurementType, 1, time.Now().UTC())
		if len(series) == 0 {
			c.JSON(http.StatusInternalServerError, APIresponse{Success: false, Message: "No data generated"})
			return
		}
		data := series[0]

		lock.Lock()
		_, exists := qosDB[id]
		if !exists {
			lock.Unlock()
			c.JSON(http.StatusNotFound, APIresponse{Success: false, Message: "Data not found"})
			return
		}
		qosDB[id] = data
		lock.Unlock()

		c.JSON(http.StatusOK, APIresponse{Success: true, Message: "QoS data updated", Data: data})
	})

	r.PATCH("/qos/:id", func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]interface{}
		if c.Request.Body != nil {
			_ = c.ShouldBindJSON(&updates)
		}

		lock.Lock()
		data, exists := qosDB[id]
		lock.Unlock()

		if !exists {
			c.JSON(http.StatusNotFound, APIresponse{Success: false, Message: "Data not found"})
			return
		}

		c.JSON(http.StatusOK, APIresponse{Success: true, Message: "QoS data patched", Data: data})
	})

	r.DELETE("/qos/:id", func(c *gin.Context) {
		id := c.Param("id")

		lock.Lock()
		_, exists := qosDB[id]
		if !exists {
			lock.Unlock()
			c.JSON(http.StatusNotFound, APIresponse{Success: false, Message: "Data not found"})
			return
		}
		delete(qosDB, id)
		lock.Unlock()

		c.JSON(http.StatusOK, APIresponse{Success: true, Message: "QoS data deleted"})
	})
	r.RunTLS(":8080", "server.crt", "server.key")
}
