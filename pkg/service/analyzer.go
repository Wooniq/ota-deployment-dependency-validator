package service

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

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

// AnalyzeAndSave: JSON 형식의 인벤토리 데이터를 파싱하여 ECU별로 정규화 후 채널에 전달함
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

// AnalyzeAndSaveBinary: DLT 규격의 바이너리 페이로드에서 SOH와 ECU 버전 정보를 정규식으로 추출함
func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	if len(payload) < 21 {
		return fmt.Errorf("데이터 길이 부족")
	}

	// SOH 추출: 페이로드 마지막 4바이트를 IEEE 754 float32(BigEndian)로 해석
	sohOffset := len(payload) - 4
	sohRaw := binary.BigEndian.Uint32(payload[sohOffset:])
	batterySOH := float64(math.Float32frombits(sohRaw))

	// ECU 정보 추출: 정규식을 활용하여 텍스트 데이터 내 ECU 명칭 및 버전 식별
	rawText := string(payload[:sohOffset])
	re := regexp.MustCompile(`(BMS|ADAS|ICU|TCU)\s+v?(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindAllStringSubmatch(rawText, -1)

	if len(matches) == 0 {
		return fmt.Errorf("정규식 매칭 실패 (Raw: %s)", rawText)
	}

	for _, match := range matches {
		ecuID := match[1]
		major, _ := strconv.Atoi(match[2])
		minor, _ := strconv.Atoi(match[3])
		patch, _ := strconv.Atoi(match[4])
		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)

		status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

		entity := repository.VehicleInfo{
			VIN:          vinFromTopic,
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

// startHanaBatchWorker: 채널에 쌓인 데이터를 모아 주기적으로 SAP HANA DB에 벌크 적재함
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

// flush: 실제 DB 리포지토리를 호출하여 데이터를 저장함
func (a *OTAAnalyzer) flush(batch []repository.VehicleInfo) {
	if err := a.Repo.BulkUpsertVehicles(batch); err != nil {
		log.Printf("[Error] HANA Bulk 적재 실패: %v", err)
	}
}

// performDeepAnalysis: 배터리 상태 및 소프트웨어 버전을 비교하여 업데이트 적격성을 판별함
func (s *OTAAnalyzer) performDeepAnalysis(adasVer, bmsVer string, soh float64) (repository.StatusCode, bool) {
	// 배터리 SOH 0.3 미만일 경우 안전을 위해 업데이트 차단
	if soh < 0.3 {
		return repository.StatusBatteryLow, false
	}

	hasLowerVersion := (adasVer < s.TargetADAS) || (bmsVer < s.TargetBMS)
	if hasLowerVersion {
		return repository.StatusIdle, true
	}
	return repository.StatusSuccess, false
}

// parseVersionParts: 버전 문자열(vX.Y.Z)을 정수형 필드로 분리함
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
