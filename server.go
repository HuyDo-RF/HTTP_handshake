package main

import (
	"fmt"
	"net/http"
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

	// --- POST: tạo chuỗi Time Series ---
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

		c.JSON(http.StatusOK, APIresponse{
			Success: true, Message: "QoS series generated", Data: series,
		})
	})

	// --- GET: lấy record theo ID ---
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

	// --- PUT: thay thế record ---
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

	// --- PATCH: cập nhật nhẹ (giữ nguyên data cũ) ---
	r.PATCH("/qos/:id", func(c *gin.Context) {
		id := c.Param("id")
		var updates map[string]interface{}
		if c.Request.Body != nil {
			_ = c.ShouldBindJSON(&updates) // không ép buộc
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

	// --- DELETE: xoá record ---
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

	// chạy HTTPS server
	r.RunTLS(":8080", "server.crt", "server.key")
}
