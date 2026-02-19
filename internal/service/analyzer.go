package service

// 수집된 데이터를 분석하여 업데이트 필요 여부 결정 (비즈니스 로직)

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Wooniq/ota-agent/internal/repository"
)

// OTAAnalyzer : 비즈니스 로직을 수행하는 서비스 객체
// Repository 인터페이스를 의존성 주입(DI) 받아 DB와 통신합니다.
type OTAAnalyzer struct {
	Repo       *repository.HANARepository
	TargetADAS string
	TargetBMS  string
}

// NewOTAAnalyzer : 분석기 생성자
func NewOTAAnalyzer(repo *repository.HANARepository, adasVer, bmsVer string) *OTAAnalyzer {
	return &OTAAnalyzer{
		Repo:       repo,
		TargetADAS: adasVer,
		TargetBMS:  bmsVer,
	}
}

// AnalyzeAndSave : 에이전트의 DTO를 분석하여 Entity로 변환 후 HANA DB에 저장
func (s *OTAAnalyzer) AnalyzeAndSave(vin, payload string) error {
	// 1. DTO 객체 생성 및 JSON 파싱
	// repository 패키지에 정의된 VehicleInventoryDTO를 사용합니다.
	var dto repository.VehicleInventoryDTO
	if err := json.Unmarshal([]byte(payload), &dto); err != nil {
		return fmt.Errorf("DTO 파싱 실패 (VIN: %s): %w", vin, err)
	}

	// 2. 비즈니스 로직: 업데이트 필요 여부 판단
	// 서버의 TargetVersion과 비교하여 결과 산출
	needsUpdate := (dto.ADAS < s.TargetADAS) || (dto.BMS < s.TargetBMS)

	// 3. Entity 변환 (DTO -> repository.VehicleInfo)
	// 에이전트에서 온 데이터에 서버 관리 데이터(LastReported, NeedsUpdate)를 결합합니다.
	entity := repository.VehicleInfo{
		VIN:          vin,
		HWVersion:    dto.HW,
		ADASVersion:  dto.ADAS,
		BMSVersion:   dto.BMS,
		UpdateStatus: "Idle", // 기본 상태값 부여
		RegionCode:   dto.Reg,
		BatterySOH:   dto.SOH,
		LastReported: time.Now(), // 서버 수신 시점 기록
		NeedsUpdate:  needsUpdate,
	}

	// 4. HANA DB 저장소 호출
	if err := s.Repo.UpsertVehicle(entity); err != nil {
		return fmt.Errorf("HANA DB 저장 실패: %w", err)
	}

	return nil
}

// AnalyzeAndSaveBinary : Autosar DLT 표준 바이너리 데이터를 해석하여 분석 및 저장
func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	// 1. [표준 규격] 바이너리 데이터 역직렬화 (Deserialization)
	// 최소 길이 체크
	if len(payload) < 17 {
		return fmt.Errorf("데이터 길이 부족")
	}

	// 1-1. VIN 추출 및 자동 보정
	rawVin := strings.TrimSpace(string(payload[0:17]))
	vin := rawVin

	// 페이로드에서 읽은 VIN이 토픽과 다를 경우(잘림 등), MQTT 토픽 정보를 신뢰함
	if vinFromTopic != vin {
		log.Printf("[Security-Alert] VIN 보정 실행: %s -> %s", vin, vinFromTopic)
		vin = vinFromTopic
	}

	// 1-2. ECU 데이터 파싱 (17번 인덱스 이후)
	ecuInventory := make(map[string]string)
	currentPos := 17

	for currentPos < len(payload) {
		// ECU ID(4자)가 존재할 수 있는 최소 길이 확인
		if currentPos+4 > len(payload) {
			break
		}

		// ECU ID 추출 (4바이트 고정)
		ecuID := strings.TrimSpace(string(payload[currentPos : currentPos+4]))
		currentPos += 4

		// SW 버전 추출 (Null 종료 문자 '\x00' 탐색)
		endOfVersion := bytes.IndexByte(payload[currentPos:], 0x00)
		var version string

		if endOfVersion == -1 {
			// Null 문자가 없으면 남은 전체를 버전으로 간주
			version = strings.TrimSpace(string(payload[currentPos:]))
			currentPos = len(payload)
		} else {
			version = strings.TrimSpace(string(payload[currentPos : currentPos+endOfVersion]))
			currentPos += endOfVersion + 1 // Null 문자 다음 위치로 이동
		}

		if ecuID != "" {
			ecuInventory[ecuID] = version
		}
	}

	// 2. 비즈니스 로직: 특정 키(ADAS, BMS) 추출 및 업데이트 필요 여부 판단
	adasVer := ecuInventory["ADAS"]
	bmsVer := ecuInventory["BMS"]
	needsUpdate := (adasVer < s.TargetADAS) || (bmsVer < s.TargetBMS)

	// 3. Entity 생성 (DB 컬럼명과 Go 구조체 필드 매핑 확인)
	entity := repository.VehicleInfo{
		VIN:          vin,
		HWVersion:    ecuInventory["HW"],
		ADASVersion:  ecuInventory["ADAS"],
		BMSVersion:   ecuInventory["BMS"],
		UpdateStatus: "Idle",
		RegionCode:   "KR", // 기본값
		BatterySOH:   85.5, // 필요 시 파싱 데이터 매핑
		LastReported: time.Now(),
		NeedsUpdate:  needsUpdate,
	}

	// 4. HANA DB 저장소 호출
	err := s.Repo.UpsertVehicle(entity)
	if err != nil {
		return fmt.Errorf("HANA DB 저장 실패 (VIN: %s): %w", vin, err)
	}

	log.Printf("[Success] VIN:%s 표준 데이터 적재 완료", vin)
	return nil
}
