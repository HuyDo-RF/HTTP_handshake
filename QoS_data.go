package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"
)

type database struct { //QoS theo chuan github
	ID                string  `json:"id"`                //ma dinh danh cho cac lan do
	DlPacketDelay     uint32  `json:"dlPacketDelay"`     //tre1
	UlPacketDelay     uint32  `json:"ulPacketDelay"`     //tre2
	RtrPacketDelay    uint32  `json:"rtrPacketDelay"`    //tre3
	MeasureFailure    bool    `json:"measureFailure"`    //co fail hay ko
	DlAveThroughput   float64 `json:"dlAveThroughput"`   //toc do tb kem don vi
	UlAveThroughput   float64 `json:"ulAveThroughput"`   //toc do tb kem don vi
	DlCongestion      int     `json:"dlCongestion"`      //do tac nghen
	UlCongestion      int     `json:"ulCongestion"`      //do tac nghen
	DefaultQosFlowInd bool    `json:"defaultQosFlowInd"` //QoS flow co mac  dinh ko, mac dinh la false

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

// Tạo oQoS Data
func GenerateQoSData(clientID, measurementType string) *database {
	id := fmt.Sprintf("qos_%d_%s", time.Now().UnixNano(), clientID)

	// Set range tuỳ theo loại measurement
	var baseDlDelay, baseUlDelay, baseRtt uint32
	var dlTputMin, dlTputMax, ulTputMin, ulTputMax float64
	var failRate float32

	switch measurementType {
	case "real-time": // giả lập URLLC
		baseDlDelay = uint32(5 + rand.Intn(5))  // 5–10 ms
		baseUlDelay = uint32(6 + rand.Intn(5))  // 6–11 ms
		baseRtt = baseDlDelay + baseUlDelay + 2 // ~15–25 ms
		dlTputMin, dlTputMax = 50, 200          // Mbps
		ulTputMin, ulTputMax = 10, 100          // Mbps
		failRate = 0.02
	case "stress-test": // giả lập eMBB tải nặng
		baseDlDelay = uint32(20 + rand.Intn(40)) // 20–60 ms
		baseUlDelay = uint32(25 + rand.Intn(40)) // 25–65 ms
		baseRtt = baseDlDelay + baseUlDelay + 10 // ~60–120 ms
		dlTputMin, dlTputMax = 20, 800           // Mbps
		ulTputMin, ulTputMax = 5, 200            // Mbps
		failRate = 0.08
	default: // mặc định eMBB bình thường
		baseDlDelay = uint32(15 + rand.Intn(20)) // 15–35 ms
		baseUlDelay = uint32(18 + rand.Intn(20)) // 18–38 ms
		baseRtt = baseDlDelay + baseUlDelay + 5  // ~40–70 ms
		dlTputMin, dlTputMax = 100, 1000         // Mbps
		ulTputMin, ulTputMax = 20, 150           // Mbps
		failRate = 0.05
	}

	// Congestion ảnh hưởng throughput & delay
	congestion := rand.Intn(100)                   // 0–99
	delayFactor := 1.0 + float64(congestion)/200.0 // delay tăng theo congestion
	tputFactor := 1.0 - float64(congestion)/300.0  // throughput giảm theo congestion

	dlDelay := uint32(float64(baseDlDelay) * delayFactor)
	ulDelay := uint32(float64(baseUlDelay) * delayFactor)
	rtt := uint32(float64(baseRtt) * delayFactor)

	dlTput := dlTputMin + rand.Float64()*(dlTputMax-dlTputMin)
	ulTput := ulTputMin + rand.Float64()*(ulTputMax-ulTputMin)
	dlTput *= tputFactor
	ulTput *= tputFactor

	return &database{
		ID:                id,
		DlPacketDelay:     dlDelay,
		UlPacketDelay:     ulDelay,
		RtrPacketDelay:    rtt,
		MeasureFailure:    rand.Float32() < failRate,
		DlAveThroughput:   dlTput,
		UlAveThroughput:   ulTput,
		DlCongestion:      congestion,
		UlCongestion:      congestion + rand.Intn(10) - 5, // dao động nhẹ
		DefaultQosFlowInd: rand.Float32() < 0.5,           // ~50% có QFI mặc định
		Timestamp:         time.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
		ClientID:          clientID,
		ServerID:          "qos-server-api-v1",
		MeasurementType:   measurementType,
	}
}

func SaveQoSDataToCSV(data *database, filename string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	record := []string{
		data.ID,
		strconv.Itoa(int(data.DlPacketDelay)),
		strconv.Itoa(int(data.UlPacketDelay)),
		strconv.Itoa(int(data.RtrPacketDelay)),
		strconv.FormatBool(data.MeasureFailure),
		strconv.FormatFloat(data.DlAveThroughput, 'f', 2, 64),
		strconv.FormatFloat(data.UlAveThroughput, 'f', 2, 64),
		strconv.Itoa(data.DlCongestion),
		strconv.Itoa(data.UlCongestion),
		strconv.FormatBool(data.DefaultQosFlowInd),
		data.MeasurementType,
		data.ClientID,
		data.Timestamp,
	}

	return writer.Write(record)
}

func CreateCSVHeader(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"ID", "DlPacketDelay", "UlPacketDelay", "RtrPacketDelay",
		"MeasureFailure", "DlAveThroughput", "UlAveThroughput",
		"DlCongestion", "UlCongestion", "DefaultQosFlowInd",
		"MeasurementType", "ClientID", "Timestamp",
	}

	return writer.Write(headers)
}
