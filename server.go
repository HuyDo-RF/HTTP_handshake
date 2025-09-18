package main

import (
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

	r.Run(":8080")
}
