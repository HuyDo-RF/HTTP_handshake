package main

import (
	"encoding/csv"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"
)

// ==================== STRUCT =====================
type database struct {
	ID                string  `json:"id"`
	DlPacketDelay     float64 `json:"DlPacketDelay"`
	UlPacketDelay     float64 `json:"UlPacketDelay"`
	RtrPacketDelay    float64 `json:"RtrPacketDelay"`
	MeasureFailure    bool    `json:"MeasureFailure"`
	DlAveThroughput   float64 `json:"DlAveThroughput"`
	UlAveThroughput   float64 `json:"UlAveThroughput"`
	DlCongestion      float64 `json:"DlCongestion"`
	UlCongestion      float64 `json:"UlCongestion"`
	DefaultQosFlowInd bool    `json:"defaultQosFlowInd"`
	Timestamp         string  `json:"Timestamp"`
	ServerID          string  `json:"serverID"`
	ClientID          string  `json:"clientID"`
	MeasurementType   string  `json:"measurementType"`
	IsAnomaly         int     `json:"isAnomaly"`
}

type QoSCreateRequest struct {
	ClientID        string `json:"clientId"`
	MeasurementType string `json:"measurementType"`
	Duration        int    `json:"duration"` // số điểm trong chuỗi
}

type APIresponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

func GenerateQoSSeriesFromStart(clientID, measurementType string, n int, start time.Time) []*database {
	records := make([]*database, 0, n)

	// drift
	var driftDl, driftUl, driftDelay float64

	for i := 0; i < n; i++ {
		t := float64(i)
		isAnomaly := 0

		// Throughput: sin + drift + noise
		driftDl += rand.NormFloat64() * 0.5
		driftUl += rand.NormFloat64() * 0.2

		dlTput := 200 + 80*math.Sin(2*math.Pi*t/100) + driftDl + rand.NormFloat64()*5
		ulTput := 200 + 30*math.Sin(2*math.Pi*t/90) + driftUl + rand.NormFloat64()*3
		if dlTput < 5 {
			dlTput = 5
		}
		if ulTput < 1 {
			ulTput = 1
		}

		//Delay: sin riêng + ngược throughput + drift
		driftDelay += rand.NormFloat64() * 0.3
		baseDlDelay := 20 + 5*math.Sin(2*math.Pi*t/110)
		dlDelay := baseDlDelay + (250-dlTput)/50 + driftDelay + rand.NormFloat64()*2
		if dlDelay < 5 {
			dlDelay = 5
		}

		baseUlDelay := 15 + 5*math.Sin(2*math.Pi*t/95)
		ulDelay := baseUlDelay + (100-ulTput)/20 + driftDelay + rand.NormFloat64()*2
		if ulDelay < 5 {
			ulDelay = 5
		}

		rtt := dlDelay + ulDelay + 5 + rand.NormFloat64()*2

		// Congestion: sin độc lập + noise Gaussian
		baseDlCong := 6.5 + 3.0*math.Sin(2*math.Pi*t/120)                // dao động chậm
		noiseDl := 0.5 * math.Sin(2*math.Pi*t/15+rand.Float64()*math.Pi) // nhiễu nhanh
		dlCong := baseDlCong + noiseDl + rand.NormFloat64()*0.3          // thêm chút noise Gaussian
		if dlCong < 3 {
			dlCong = 3 + rand.Float64()*0.5 // tránh vượt dưới 3%
		} else if dlCong > 10 {
			dlCong = 10 - rand.Float64()*0.5
		}

		baseUlCong := 6.0 + 3.0*math.Sin(2*math.Pi*t/130+math.Pi/4)
		noiseUl := 0.5 * math.Sin(2*math.Pi*t/20+rand.Float64()*math.Pi/3)
		ulCong := baseUlCong + noiseUl + rand.NormFloat64()*0.3
		if ulCong < 3 {
			ulCong = 3 + rand.Float64()*0.5
		} else if ulCong > 10 {
			ulCong = 10 - rand.Float64()*0.5
		}

		// Anomaly
		if rand.Float64() < 0.01 {
			isAnomaly = 1

			switch rand.Intn(4) {
			case 0: // congestion spike
				dlCong += 20
				ulCong += 20
			case 1: // delay spike
				dlDelay *= 1.5
				ulDelay *= 1.5
			case 2: // throughput drop
				dlTput *= 0.2
				ulTput *= 0.2
			case 3: // combination
				dlTput *= 0.3
				dlDelay *= 1.5
				dlCong += 20
			}
		}

		//Tạo record
		d := &database{
			ID:                fmt.Sprintf("qos_%d_%s", time.Now().UnixNano(), clientID),
			DlPacketDelay:     dlDelay,
			UlPacketDelay:     ulDelay,
			RtrPacketDelay:    rtt,
			MeasureFailure:    rand.Float32() < 0.05,
			DlAveThroughput:   dlTput,
			UlAveThroughput:   ulTput,
			DlCongestion:      dlCong,
			UlCongestion:      ulCong,
			DefaultQosFlowInd: rand.Float32() < 0.5,
			Timestamp:         start.Add(time.Duration(i) * time.Minute).Format("2006-01-02T15:04:05Z"),
			ClientID:          clientID,
			ServerID:          "qos-server-api-v1",
			MeasurementType:   measurementType,
			IsAnomaly:         isAnomaly,
		}
		records = append(records, d)
	}
	return records
}

// CSV
func SaveQoSSeriesToCSV(data []*database, filename string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, r := range data {
		record := []string{
			r.ID,
			strconv.FormatFloat(r.DlPacketDelay, 'f', 2, 64),
			strconv.FormatFloat(r.UlPacketDelay, 'f', 2, 64),
			strconv.FormatFloat(r.RtrPacketDelay, 'f', 2, 64),
			strconv.FormatBool(r.MeasureFailure),
			strconv.FormatFloat(r.DlAveThroughput, 'f', 2, 64),
			strconv.FormatFloat(r.UlAveThroughput, 'f', 2, 64),
			strconv.FormatFloat(r.DlCongestion, 'f', 2, 64),
			strconv.FormatFloat(r.UlCongestion, 'f', 2, 64),
			strconv.FormatBool(r.DefaultQosFlowInd),
			r.MeasurementType,
			r.ClientID,
			r.Timestamp,
			strconv.Itoa(r.IsAnomaly),
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
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
		"MeasurementType", "ClientID", "Timestamp", "IsAnomaly",
	}
	return writer.Write(headers)
}
