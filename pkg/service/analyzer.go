package service

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/Wooniq/ota-agent/pkg/repository"
)

type OTAAnalyzer struct {
	Repo       *repository.HANARepository
	TargetADAS string
	TargetBMS  string
	dataChan   chan repository.VehicleInfo
	batchSize  int
}

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

// AnalyzeAndSave : JSON 데이터를 정규화하여 ECU별로 개별 Row 생성
func (s *OTAAnalyzer) AnalyzeAndSave(vin, payload string) error {
	var dto repository.VehicleInventoryDTO
	if err := json.Unmarshal([]byte(payload), &dto); err != nil {
		return fmt.Errorf("DTO 파싱 실패 (VIN: %s): %w", vin, err)
	}

	// [정규화 로직] ADAS와 BMS 데이터를 각각 별도의 엔티티로 분리
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
		// 배치 워커 채널로 전송
		s.dataChan <- entity
	}
	return nil
}

// AnalyzeAndSaveBinary : DLT 패킷 해석 및 LittleEndian SOH 추출
func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	if len(payload) < 21 {
		return fmt.Errorf("데이터 길이 부족")
	}

	// 1. [SOH 추출] LittleEndian 적용하여 수치 정상화
	sohOffset := len(payload) - 4
	sohRaw := binary.LittleEndian.Uint32(payload[sohOffset:])
	batterySOH := float64(math.Float32frombits(sohRaw))

	// 2. [ECU 데이터 파싱]
	ecuInventory := make(map[string]string)
	currentPos := 17
	limitPos := len(payload) - 4

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

	// 3. [정규화 적재] 추출된 데이터를 ECU별로 쪼개서 채널에 삽입
	for ecuID, ver := range ecuInventory {
		if ecuID == "HW" {
			continue
		} // HW 정보는 메타데이터로 활용

		status, needsUpdate := s.performDeepAnalysis(ver, ver, batterySOH)
		major, minor, patch := parseVersionParts(ver)

		entity := repository.VehicleInfo{
			VIN:          vinFromTopic,
			ECUType:      ecuID,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			HWVersion:    ecuInventory["HW"],
			BatterySOH:   batterySOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}
		s.dataChan <- entity
	}

	log.Printf("[Success] VIN:%s 정규화 분석 완료 (SOH: %.2f)", vinFromTopic, batterySOH)
	return nil
}

// startHanaBatchWorker : 주기적으로 벌크 저장 수행
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

func (a *OTAAnalyzer) flush(batch []repository.VehicleInfo) {
	if err := a.Repo.BulkUpsertVehicles(batch); err != nil {
		log.Printf("[Error] HANA Bulk 적재 실패: %v", err)
	}
}

func (s *OTAAnalyzer) performDeepAnalysis(adasVer, bmsVer string, soh float64) (repository.StatusCode, bool) {
	if soh < 0.3 {
		return repository.StatusBatteryLow, false
	}
	hasLowerVersion := (adasVer < s.TargetADAS) || (bmsVer < s.TargetBMS)
	if hasLowerVersion {
		return repository.StatusIdle, true
	}
	return repository.StatusSuccess, false
}

// parseVersionParts : "v2.3.5" -> 2, 3, 5 변환 헬퍼
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
