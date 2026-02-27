package service

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"regexp"
	"strconv"
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

func (a *OTAAnalyzer) startHanaBatchWorker() {
	var batch []repository.VehicleInfo
	ticker := time.NewTicker(3 * time.Second)
	log.Println("[Service] HANA DB Batch Worker 가동 시작")

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

func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	if len(payload) < 21 {
		return fmt.Errorf("데이터 길이 부족")
	}

	// 1. [SOH 추출] 마지막 4바이트 바이너리
	sohOffset := len(payload) - 4
	sohRaw := binary.BigEndian.Uint32(payload[sohOffset:])
	batterySOH := float64(math.Float32frombits(sohRaw))

	// 2. [페이로드 분석] 현장 텍스트 데이터 (DLT)
	rawText := string(payload[:sohOffset])
	
	// 현장 데이터 패턴 매칭 (ECU명 vMajor.Minor.Patch)
	re := regexp.MustCompile(`(BMS|ADAS|ICU|TCU|TCU)\s+v?(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindAllStringSubmatch(rawText, -1)

	if len(matches) == 0 {
		return fmt.Errorf("패턴 매칭 실패: %s", rawText)
	}

	for _, match := range matches {
		ecuID := match[1]
		major, _ := strconv.Atoi(match[2])
		minor, _ := strconv.Atoi(match[3])
		patch, _ := strconv.Atoi(match[4])

		status, needsUpdate := s.performDeepAnalysis(match[0], match[0], batterySOH)

		entity := repository.VehicleInfo{
			VIN:          vinFromTopic,
			ECUType:      ecuID,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			BatterySOH:   batterySOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}
		s.dataChan <- entity
	}
	return nil
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

func (a *OTAAnalyzer) flush(batch []repository.VehicleInfo) {
	if err := a.Repo.BulkUpsertVehicles(batch); err != nil {
		log.Printf("[Error] HANA Bulk 적재 실패: %v", err)
		return
	}
	log.Printf("[HANA] %d건의 차량 데이터 배치 적재 성공", len(batch))
}
