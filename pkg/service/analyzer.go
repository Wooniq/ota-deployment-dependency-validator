package service

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/Wooniq/ota-agent/pkg/protocol"
	"github.com/Wooniq/ota-agent/pkg/repository"
)

// OTAAnalyzer: 차량 데이터 분석 및 SAP HANA DB 적재를 담당하는 서비스 객체
type OTAAnalyzer struct {
	Repo       *repository.HANARepository
	TargetADAS string
	TargetBMS  string
	dataChan   chan repository.VehicleInfo // 비동기 DB 적재를 위한 데이터 채널
	batchSize  int                         // 벌크 Insert 단위
}

// NewOTAAnalyzer: 분석기 인스턴스를 생성하고 백그라운드 배치 워커를 가동함
func NewOTAAnalyzer(repo *repository.HANARepository, adasVer, bmsVer string) *OTAAnalyzer {
	a := &OTAAnalyzer{
		Repo:       repo,
		TargetADAS: adasVer,
		TargetBMS:  bmsVer,
		dataChan:   make(chan repository.VehicleInfo, 2000),
		batchSize:  100,
	}
	go a.startHanaBatchWorker()
	return a
}

// AnalyzeAndSave: JSON 형식의 데이터를 파싱하여 ECU별로 정규화 후 채널에 전달함
func (s *OTAAnalyzer) AnalyzeAndSave(vin, payload string) error {
	var dto repository.VehicleInventoryDTO
	if err := json.Unmarshal([]byte(payload), &dto); err != nil {
		return fmt.Errorf("JSON 파싱 실패 (VIN: %s): %w", vin, err)
	}

	ecuTargets := []struct {
		ID  string
		Ver string
	}{
		{"ADAS", dto.ADAS},
		{"BMS", dto.BMS},
	}

	for _, ecu := range ecuTargets {
		status, needsUpdate := s.performDeepAnalysis(ecu.Ver, ecu.Ver, dto.SOH)
		major, minor, patch := parseVersionParts(ecu.Ver)

		entity := repository.VehicleInfo{
			VIN:          vin,
			ECUType:      ecu.ID,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			HWVersion:    dto.HW,
			BatterySOH:   dto.SOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}
		s.dataChan <- entity
	}
	return nil
}

/*// AnalyzeAndSaveBinary: 고도화된 고정폭 바이너리 규격에 맞춰 데이터를 파싱함
func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	// [규격] VIN(17) + SOH(4) + (ECU_ID(4) + Maj(1) + Min(1) + Pat(1)) * N
	if len(payload) < 21 {
		return fmt.Errorf("데이터 길이 부족")
	}

	// 1. VIN 추출 (0~17): 공백 제거 후 실제 값 확보
	vin := strings.TrimSpace(string(payload[0:17]))

	// 2. SOH 추출 (17~21): BigEndian IEEE 754 float32
	sohRaw := binary.BigEndian.Uint32(payload[17:21])
	batterySOH := float64(math.Float32frombits(sohRaw))

	// 3. ECU 정보 반복 추출 (21 index 부터 끝까지)
	// 한 세트당 7바이트 (ID 4 + Ver 3)
	for offset := 21; offset+7 <= len(payload); offset += 7 {
		// ECU ID 추출 (4바이트)
		ecuID := strings.TrimSpace(string(payload[offset : offset+4]))
		if ecuID == "" {
			continue
		}

		// 버전 추출 (3바이트 정수 -> 문자열 복원)
		major := int(payload[offset+4])
		minor := int(payload[offset+5])
		patch := int(payload[offset+6])
		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)

		// 분석 및 엔티티 생성
		status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

		entity := repository.VehicleInfo{
			VIN:          vin, // 혹은 vinFromTopic
			ECUType:      ecuID,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			HWVersion:    "HW_REV_01",
			BatterySOH:   batterySOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}
		s.dataChan <- entity
	}

	return nil
}
*/

func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	// 1. DLT 패킷 해석 (16바이트 헤더 제거 및 검증)
	// protocol 패키지에 이미 구현된 ParseDltPacket을 사용하여 순수 데이터만 추출합니다.
	pureData, err := protocol.ParseDltPacket(payload)
	if err != nil {
		return fmt.Errorf("DLT 해석 실패: %v", err)
	}

	// 2. 최소 데이터 길이 확인 (VIN 17 + SOH 4 = 21바이트)
	if len(pureData) < 21 {
		return fmt.Errorf("순수 데이터 길이 부족")
	}

	// 3. VIN 추출 (0~17) - pureData 기준
	vin := strings.TrimSpace(string(pureData[0:17]))

	// 4. SOH 추출 (17~21) - pureData 기준
	sohRaw := binary.BigEndian.Uint32(pureData[17:21])
	batterySOH := float64(math.Float32frombits(sohRaw))

	// 5. ECU 정보 반복 추출 (21 index 부터 시작)
	for offset := 21; offset+7 <= len(pureData); offset += 7 {
		ecuID := strings.TrimSpace(string(pureData[offset : offset+4]))
		if ecuID == "" {
			continue
		}

		major := int(pureData[offset+4])
		minor := int(pureData[offset+5])
		patch := int(pureData[offset+6])
		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)

		// 분석 및 엔티티 생성
		status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

		entity := repository.VehicleInfo{
			VIN:          vin,
			ECUType:      ecuID,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			HWVersion:    "HW_REV_01",
			BatterySOH:   batterySOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}
		s.dataChan <- entity
	}

	return nil
}

// startHanaBatchWorker: 채널 데이터를 모아 주기적으로 SAP HANA DB에 벌크 적재함
func (a *OTAAnalyzer) startHanaBatchWorker() {
	var batch []repository.VehicleInfo
	ticker := time.NewTicker(3 * time.Second)

	for {
		select {
		case entity := <-a.dataChan:
			batch = append(batch, entity)
			if len(batch) >= a.batchSize {
				a.flush(batch)
				batch = nil
			}
		case <-ticker.C:
			if len(batch) > 0 {
				a.flush(batch)
				batch = nil
			}
		}
	}
}

// flush: 리포지토리를 호출하여 메모리 상의 배치를 실제 DB에 UPSERT 함
func (a *OTAAnalyzer) flush(batch []repository.VehicleInfo) {
	if err := a.Repo.BulkUpsertVehicles(batch); err != nil {
		log.Printf("[Error] HANA Bulk 적재 실패: %v", err)
	}
}

// performDeepAnalysis: 배터리 상태 및 버전을 비교하여 업데이트 적격성을 판별함
func (s *OTAAnalyzer) performDeepAnalysis(adasVer, bmsVer string, soh float64) (repository.StatusCode, bool) {
	// 배터리 SOH 0.3 미만일 경우 안전을 위해 업데이트 대상에서 제외
	if soh < 0.3 {
		return repository.StatusBatteryLow, false
	}

	hasLowerVersion := (adasVer < s.TargetADAS) || (bmsVer < s.TargetBMS)
	if hasLowerVersion {
		return repository.StatusIdle, true
	}
	return repository.StatusSuccess, false
}

// parseVersionParts: 버전 문자열(vX.Y.Z)을 정수 필드로 분리함
func parseVersionParts(v string) (int, int, int) {
	v = strings.TrimPrefix(v, "v")
	p := strings.Split(v, ".")
	if len(p) < 3 {
		return 0, 0, 0
	}
	var res [3]int
	for i := 0; i < 3; i++ {
		fmt.Sscanf(p[i], "%d", &res[i])
	}
	return res[0], res[1], res[2]
}
