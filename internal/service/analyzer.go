package service

// 수집된 데이터를 분석하여 업데이트 필요 여부 결정 (비즈니스 로직)

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
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

// AnalyzeAndSave : JSON DTO 분석 및 상세 상태 코드 적용
func (s *OTAAnalyzer) AnalyzeAndSave(vin, payload string) error {
	var dto repository.VehicleInventoryDTO
	if err := json.Unmarshal([]byte(payload), &dto); err != nil {
		return fmt.Errorf("DTO 파싱 실패 (VIN: %s): %w", vin, err)
	}

	// [ADR 0002] 상세 분석
	status, needsUpdate := s.performDeepAnalysis(dto.ADAS, dto.BMS, dto.SOH)

	entity := repository.VehicleInfo{
		VIN:          vin,
		HWVersion:    dto.HW,
		ADASVersion:  dto.ADAS,
		BMSVersion:   dto.BMS,
		UpdateStatus: status, // 상세 코드(00, E1 등) 부여
		RegionCode:   dto.Reg,
		BatterySOH:   dto.SOH,
		LastReported: time.Now(),
		NeedsUpdate:  needsUpdate,
	}

	if err := s.Repo.UpsertVehicle(entity); err != nil {
		return fmt.Errorf("HANA DB 저장 실패: %w", err)
	}

	return nil
}

// AnalyzeAndSaveBinary : DLT 패킷을 해석하고 SOH 데이터를 정밀 추출하여 분석 및 저장
func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	// 1. 최소 길이 체크 (VIN 17B + 최소 ECU 데이터)
	if len(payload) < 21 { // VIN(17) + SOH(4) 기준 최소 길이
		return fmt.Errorf("데이터 길이 부족")
	}

	// 1-1. SOH 추출 (페이로드의 가장 마지막 4바이트에 float32로 위치한다고 가정)
	// DLT 표준에 맞춰 BigEndian 방식으로 해석
	sohOffset := len(payload) - 4
	sohRaw := binary.BigEndian.Uint32(payload[sohOffset:])
	batterySOH := float64(math.Float32frombits(sohRaw)) // 4바이트 바이너리를 float64로 변환

	// 1-2. VIN 추출 (0~17바이트)
	vin := strings.TrimSpace(string(payload[0:17]))
	if vinFromTopic != vin {
		log.Printf("[Security] VIN 보정: %s -> %s", vin, vinFromTopic)
		vin = vinFromTopic
	}

	// 1-3. ECU 데이터 파싱 (17번 인덱스부터 SOH 시작 전까지)
	ecuInventory := make(map[string]string)
	currentPos := 17
	limitPos := len(payload) - 4 // SOH 데이터 시작 직전까지가 ECU 정보

	for currentPos < limitPos {
		if currentPos+4 > limitPos {
			break
		}
		ecuID := strings.TrimSpace(string(payload[currentPos : currentPos+4]))
		currentPos += 4

		endOfVersion := bytes.IndexByte(payload[currentPos:limitPos], 0x00)
		var version string
		if endOfVersion == -1 {
			version = strings.TrimSpace(string(payload[currentPos:limitPos]))
			currentPos = limitPos
		} else {
			version = strings.TrimSpace(string(payload[currentPos : currentPos+endOfVersion]))
			currentPos += endOfVersion + 1
		}
		if ecuID != "" {
			ecuInventory[ecuID] = version
		}
	}

	// 2. [ADR 0002] 상세 분석 수행 (추출한 batterySOH 사용)
	status, needsUpdate := s.performDeepAnalysis(ecuInventory["ADAS"], ecuInventory["BMS"], batterySOH)

	// 3. Entity 생성 및 저장
	entity := repository.VehicleInfo{
		VIN:          vin,
		HWVersion:    ecuInventory["HW"],
		ADASVersion:  ecuInventory["ADAS"],
		BMSVersion:   ecuInventory["BMS"],
		UpdateStatus: status,
		RegionCode:   "KR",
		BatterySOH:   batterySOH, // 파싱된 실측 데이터 적재
		LastReported: time.Now(),
		NeedsUpdate:  needsUpdate,
	}

	if err := s.Repo.UpsertVehicle(entity); err != nil {
		return fmt.Errorf("HANA DB 저장 실패: %w", err)
	}

	log.Printf("[Success] VIN:%s 분석 및 HANA DB 저장 완료 (Status:%s)", vin, status)
	return nil
}

// performDeepAnalysis : 전압 상태 및 버전 비교를 통한 상세 분석
func (s *OTAAnalyzer) performDeepAnalysis(adasVer, bmsVer string, soh float64) (repository.StatusCode, bool) {
	// 1. 배터리 전압(SOH) 체크 - 최우선 순위
	// 0.3(30%) 미만인 경우 업데이트를 수행할 수 없는 위험 상태(E1)로 판별합니다.
	if soh < 0.3 {
		return repository.StatusBatteryLow, false // 전압 부족 시 업데이트 대상에서 제외
	}

	// 2. 버전 비교 분석
	hasLowerVersion := (adasVer < s.TargetADAS) || (bmsVer < s.TargetBMS)

	// 3. 최종 상태 결정
	if hasLowerVersion {
		// 버전이 낮고 전압이 정상이면 업데이트 필요 대상으로 분류
		return repository.StatusIdle, true
	}

	// 최신 버전인 경우
	return repository.StatusSuccess, false
}
