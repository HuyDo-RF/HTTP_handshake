package main

import (
	"fmt"
	"math/rand"
	"time"
)

type database struct { //QoS theo chuan github
	ID                string `json:"id"`                //ma dinh danh cho cac lan do
	DlPacketDelay     uint32 `json:"dlPacketDelay"`     //tre1
	UlPacketDelay     uint32 `json:"ulPacketDelay"`     //tre2
	RtrPacketDelay    uint32 `json:"rtrPacketDelay"`    //tre3
	MeasureFailure    bool   `json:"measureFailure"`    //co fail hay ko
	DlAveThroughput   string `json:"dlAveThroughput"`   //toc do tb kem don vi
	UlAveThroughput   string `json:"ulAveThroughput"`   //toc do tb kem don vi
	DlCongestion      int    `json:"dlCongestion"`      //do tac nghen
	UlCongestion      int    `json:"ulCongestion"`      //do tac nghen
	DefaultQosFlowInd bool   `json:"defaultQosFlowInd"` //QoS flow co mac  dinh ko, mac dinh la false

	//MetaData
	Timestamp       string `json:"timestamp"`       //Thoi gian bat dau test
	ServerID        string `json:"serverID"`        //ID server
	ClientID        string `json:"clientID"`        //ID client
	MeasurementType string `json:"measurementType"` //Loai do (real-time, stress - test,...)
}

// APIresponse chuan Restful
type APIresponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

// QoSCreateRequest cho POST endpoint
type QoSCreateRequest struct {
	ClientID        string `json:"clientId"`
	MeasurementType string `json:"measurementType"`
	Duration        int    `json:"duration"` // seconds
}

// Ta oQoS Data
func GenerateQoSData(clientID, measurementType string) *database {
	id := fmt.Sprintf("qos_%d_%s", time.Now().UnixNano(), clientID)

	return &database{
		ID:                id,
		DlPacketDelay:     uint32(36 + rand.Intn(45)),
		UlPacketDelay:     uint32(36 + rand.Intn(42)),
		RtrPacketDelay:    uint32(36 + rand.Intn(85)),
		MeasureFailure:    rand.Float32() < 0.36,
		DlAveThroughput:   fmt.Sprintf("%.1f Mbps", 36+rand.Float64()*170),
		UlAveThroughput:   fmt.Sprintf("%.1f Mbps", 36+rand.Float64()*90),
		DlCongestion:      rand.Intn(36),
		UlCongestion:      rand.Intn(36),
		DefaultQosFlowInd: rand.Float32() < 36,
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		ClientID:          clientID,
		ServerID:          "qos-server-api-v1",
		MeasurementType:   measurementType,
	}
}
