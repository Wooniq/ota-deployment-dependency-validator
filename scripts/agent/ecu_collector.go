package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ECU 정보를 담는 구조체
type ECUInfo struct {
	VIN        string `json:"vin"`
	ECUType    string `json:"ecu_type"`
	HWVersion  string `json:"hw_version"`
	SWMajor    int    `json:"sw_major_v"`
	SWMinor    int    `json:"sw_minor_v"`
	SWPatch    int    `json:"sw_patch_v"`
}

func main() {
	// 1. 실제 차량이라면 /etc/ecu_info 등에서 파일을 읽겠지만,
	// 여기서는 예시 데이터를 생성함
	data := ECUInfo{
		VIN:       "KMHGN7HG1PA123456",
		ECUType:   "BMS",
		HWVersion: "HW_1.0",
		SWMajor:   2,
		SWMinor:   1,
		SWPatch:   0,
	}

	// 2. JSON 변환
	jsonData, _ := json.Marshal(data)

	// 3. OTA 서버로 전송 (FastAPI 주소)
	url := "http://localhost:8000/api/inventory/report"
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		fmt.Printf("Error reporting to server: %s\n", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("[%s] Vehicle info reported. Status: %s\n", time.Now().Format("15:04:05"), resp.Status)
}