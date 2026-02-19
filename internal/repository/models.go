package repository

import "time"

/* DB 저장 모델 (Entity)
 * 서버가 분석한 결과(NeedsUpdate), 수신 시각(LastReported) 등 서버가 관리해야 할 메타데이터 포함
 * VehicleInfo : SAP HANA DB의 VEHICLE_ECU_INVENTORY 테이블과 매핑되는 엔티티
 */

type VehicleInfo struct {
	// [식별자] ISO 3779 표준에 따른 차량 고유 식별 번호 (Primary Key)
	VIN string `json:"vin"`

	// [하드웨어] 제어기(ECU)의 물리적 하드웨어 리비전.
	// 소프트웨어 배포 시 호환성 검증(Compatibility Check)의 핵심 기준
	HWVersion string `json:"hw_version"`

	// [소프트웨어] ADAS(첨단 운전자 보조 시스템) 제어기의 현재 펌웨어 버전
	ADASVersion string `json:"adas_version"`

	// [소프트웨어] BMS(배터리 관리 시스템) 제어기의 현재 펌웨어 버전
	BMSVersion string `json:"bms_version"`

	// [운영 상태] OTA 프로세스 라이프사이클 관리.
	// (예: Idle, Downloading, Verifying, Installing, Success, Failed)
	UpdateStatus string `json:"update_status"`

	// [정책 관리] 차량의 출고 국가 또는 법규 적용 지역 코드.
	// 지역별 차등 업데이트(Differential Deployment) 및 법규 준수 확인용
	RegionCode string `json:"region_code"`

	// [안전 점검] 배터리 건강 상태 (State of Health, 0.0~1.0).
	// 업데이트 중 전력 차단 사고를 방지하기 위해 최소 요구 SOH를 검증하는 지표
	BatterySOH float64 `json:"battery_soh"`

	// [메타데이터] 차량 에이전트로부터 마지막으로 인벤토리 보고를 받은 시각.
	// 차량의 통신 상태(Alive) 및 데이터 최신성을 판단하는 지표
	LastReported time.Time `json:"last_reported"`

	// [비즈니스 로직] 목표 버전(Target) 대비 현재 버전의 업데이트 필요 여부.
	// 백엔드 분석 엔진(Analyzer)에 의해 계산된 결과값
	NeedsUpdate bool `json:"needs_update"`
}
