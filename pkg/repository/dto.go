package repository

/*
 * 에이전트 전송 데이터 (DTO): 차량에서 서버로 보내는 '날것'의 데이터
 * 통신 비용 절감을 위해 최소한의 정보만 담음
 * VehicleInventoryDTO : 에이전트가 전송하는 Raw 데이터 포맷 (Data Transfer Object)
 */

type VehicleInventoryDTO struct {
	HW   string  `json:"hw"`     // 에이전트 보고 HW 버전
	ADAS string  `json:"adas"`   // 에이전트 보고 ADAS 버전
	BMS  string  `json:"bms"`    // 에이전트 보고 BMS 버전
	SOH  float64 `json:"soh"`    // 에이전트 보고 배터리 SOH
	Reg  string  `json:"region"` // 에이전트 보고 지역 정보
}
