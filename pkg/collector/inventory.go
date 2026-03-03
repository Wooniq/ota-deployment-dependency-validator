// /config/ecu_info.json 설정 파일을 읽고 내부 구조체로 정형화하는 역할

package collector

import (
	"encoding/json"
	"errors"
	"log"
	"os"
)

// ECUInfo: 제어기 개별 정보를 담는 구조체
// json 태그를 통해 설정 파일의 필드명과 Go 구조체 필드 매핑
type ECUInfo struct {
	ID        string `json:"id"`         // 제어기 식별자 (예: BMS, BCM)
	HWVersion string `json:"hw_version"` // 하드웨어 버전
	SWVersion string `json:"sw_version"` // 현재 소프트웨어 버전
}

// VehicleInventory: 차량 전체의 식별 정보와 장착된 제어기 목록을 정의합니다.
// 1:N 관계(차량 1대 - 여러 제어기)를 표현하기 위한 최상위 구조체입니다.
type VehicleInventory struct {
	VIN  string    `json:"vin"`  // 차대번호 (Vehicle Identification Number)
	ECUs []ECUInfo `json:"ecus"` // 해당 차량에 탑재된 제어기 배열 (슬라이스)
	SOH  float64   `json:"soh"`  // 배터리 건강 상태 정보 필드
}

// [Advanced] 업데이트 가능 최소 SOH 상구 설정
const MinUpdateSOH = 0.7

// 데이터 무결성 검증 메서드
func (v *VehicleInventory) Validate() error {
	if v.VIN == "" {
		return errors.New("차대번호(VIN) 누락")
	}
	// 수집 단계에서 즉시 업데이트 적격성 1차 판단
	if v.SOH < MinUpdateSOH {
		// 전송은 하되, 상태값에 경고를 포함하거나 로깅 처리
		log.Printf("[Warn] VIN:%s SOH 부족 (%.2f). 업데이트 제한 대상", v.VIN, v.SOH)
	}
	return nil
}

// LoadInventory: JSON 파일을 읽어 SOH를 포함한 전체 인벤토리를 로드합니다.
func LoadInventory(path string) (*VehicleInventory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var inv VehicleInventory
	if err := json.Unmarshal(data, &inv); err != nil {
		return nil, err
	}

	if err := inv.Validate(); err != nil {
		return nil, err
	}

	return &inv, nil
}
