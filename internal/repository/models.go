package repository

import "time"

// StatusCode: OTA 프로세스 라이프사이클 및 상세 에러 코드를 관리하는 전용 타입
type StatusCode string

const (
	// 기본 라이프사이클 상태
	StatusIdle        StatusCode = "IDLE"
	StatusDownloading StatusCode = "DOWNLOADING"
	StatusInstalling  StatusCode = "INSTALLING"
	StatusSuccess     StatusCode = "SUCCESS"
	StatusFailed      StatusCode = "FAILED"

	// [ADR 0001] 분석 엔진 부여 상세 에러 코드
	StatusBatteryLow  StatusCode = "E1" // 전압 부족 (Battery SOH Low)
	StatusConnTimeout StatusCode = "E2" // 통신 타임아웃
	StatusDepMismatch StatusCode = "E3" // HW/SW 의존성 불일치
)

type VehicleInfo struct {
	// [식별자] ISO 3779 표준에 따른 차량 고유 식별 번호 (Primary Key)
	VIN string `json:"vin"`

	// [하드웨어] 제어기(ECU)의 물리적 하드웨어 리비전. 호환성 검증의 핵심 기준
	HWVersion string `json:"hw_version"`

	// [소프트웨어] ADAS 제어기의 현재 펌웨어 버전
	ADASVersion string `json:"adas_version"`

	// [소프트웨어] BMS 제어기의 현재 펌웨어 버전
	BMSVersion string `json:"bms_version"`

	// [운영 상태] OTA 프로세스 라이프사이클 관리 (Enum 타입 적용)
	UpdateStatus StatusCode `json:"update_status"`

	// [정책 관리] 지역별 차등 업데이트 및 법규 준수 확인용 지역 코드
	RegionCode string `json:"region_code"`

	// [안전 점검] 배터리 건강 상태 (0.0~1.0). 전력 차단 사고 방지 지표
	BatterySOH float64 `json:"battery_soh"`

	// [메타데이터] 차량 에이전트로부터 마지막 보고를 받은 시각
	LastReported time.Time `json:"last_reported"`

	// [비즈니스 로직] 목표 버전 대비 업데이트 필요 여부 (Analyzer 계산 결과)
	NeedsUpdate bool `json:"needs_update"`
}
