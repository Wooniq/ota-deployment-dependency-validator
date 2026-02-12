// /config/ecu_info.json 설정 파일을 읽고 내부 구조체로 정형화하는 역할

package collector

import (
	"encoding/json"
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
	VIN  string `json:"vin"`  // 차대번호 (Vehicle Identification Number)
	ECUs []ECU  `json:"ecus"` // 해당 차량에 탑재된 제어기 배열 (슬라이스)
}

// 데이터 무결성 검증 메서드
func (v *VehicleInventory) Validate() error {
	if v.VIN == "" {
		return errors.New("차대번호(VIN)가 누락되었습니다")
	}
	if len(v.ECUs) == 0 {
		return errors.New("장착된 제어기(ECU) 정보가 없습니다")
	}
	return nil
}

// LoadInventory: 지정된 경로의 JSON 파일을 읽어 데이터 구조체로 변환합니다.
// 파일 시스템에서 동적으로 정보를 수집하는 핵심 함수입니다.
func LoadInventory(path string) (*VehicleInventory, error) {
	// 1. 파일 읽기 (바이트 배열로)
	// 1. 파일 읽기
    data, err := os.ReadFile(path)
    if err != nil {
       return nil, err
    }

    // 2. 구조체 변수 먼저 생성
    var inv VehicleInventory

    // 3. JSON 데이터를 구조체에 채우기
    if err := json.Unmarshal(data, &inv); err != nil {
       return nil, err
    }

    // 4. 다 채워진 데이터 검사하기
    if err := inv.Validate(); err != nil {
       return nil, err
    }

    return &inv, nil
}