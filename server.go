package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

var (
	qosDB = make(map[string]*database)
	lock  sync.Mutex
)

func main() {
	r := gin.Default()

	r.POST("/qos", func(c *gin.Context) {
		var req QoSCreateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, APIresponse{Success: false, Message: "Invalid request", Error: err.Error()})
			return
		}

		data := GenerateQoSData(req.ClientID, req.MeasurementType)

		lock.Lock()
		qosDB[data.ID] = data
		lock.Unlock()

		if err := SaveQoSDataToCSV(data, "qos_training_data.csv"); err != nil {
			fmt.Printf("Error saving to CSV: %v\n", err)
		}

		c.JSON(http.StatusOK, APIresponse{Success: true, Message: "QoS data generated", Data: data})
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
		data := GenerateQoSData(req.ClientID, req.MeasurementType)
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
		if err := c.ShouldBindJSON(&updates); err != nil {
			c.JSON(http.StatusBadRequest, APIresponse{Success: false, Message: "Invalid request", Error: err.Error()})
			return
		}
		lock.Lock()
		data, exists := qosDB[id]
		if !exists {
			lock.Unlock()
			c.JSON(http.StatusNotFound, APIresponse{Success: false, Message: "Data not found"})
			return
		}
		lock.Unlock()
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
