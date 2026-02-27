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
    // [식별자] 차량 고유 식별 번호 (HANA DB Primary Key)
    VIN string `json:"vin"`

    // [제어기 식별] 어떤 ECU에 대한 정보인지 구분 (BMS, ADAS, TCU 등)
    ECUType string `json:"ecu_type"` // 추가됨

    // [소프트웨어 상세 버전] Semantic Versioning (현업 표준)
    SWMajor int `json:"sw_major"` // 추가됨 (v2.3.5에서 2)
    SWMinor int `json:"sw_minor"` // 추가됨 (v2.3.5에서 3)
    SWPatch int `json:"sw_patch"` // 추가됨 (v2.3.5에서 5)

    // [하드웨어] 제어기(ECU)의 물리적 하드웨어 리비전
    HWVersion string `json:"hw_version"`

    // [운영 상태] OTA 프로세스 라이프사이클 관리
    UpdateStatus StatusCode `json:"update_status"`

    // [정책 관리] 지역별 차등 업데이트 확인용
    RegionCode string `json:"region_code"`

    // [안전 점검] 배터리 건강 상태 (바이너리 정밀 추출 데이터)
    BatterySOH float64 `json:"battery_soh"`

    // [메타데이터] 차량 에이전트로부터 마지막 보고를 받은 시각
    LastReported time.Time `json:"last_reported"`

    // [비즈니스 로직] 업데이트 필요 여부
    NeedsUpdate bool `json:"needs_update"`
}
