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

func (s *OTAAnalyzer) AnalyzeAndSave(vin string, payload string) error {
	// 1. JSON 구조에 맞는 임시 구조체 정의 (현장 데이터 기반)
	var data struct {
		VIN  string  `json:"vin"`
		SOH  float64 `json:"soh"`
		ECUs []struct {
			ID        string `json:"id"`
			HWVersion string `json:"hw_version"`
			SWVersion string `json:"sw_version"`
		} `json:"ecus"`
	}

	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return fmt.Errorf("JSON 파싱 실패: %w", err)
	}

	// 2. 파싱된 데이터를 루프 돌며 DB 엔티티로 변환
	for _, ecu := range data.ECUs {
		cleanedID := strings.TrimSpace(ecu.ID) // "BMS " -> "BMS" 공백 제거

		// 버전 분리 (v2.3.7 -> 2, 3, 7)
		major, minor, patch := parseVersionParts(ecu.SWVersion)

		// 분석 로직 수행
		status, needsUpdate := s.performDeepAnalysis(ecu.SWVersion, ecu.SWVersion, data.SOH)

		entity := repository.VehicleInfo{
			VIN:          data.VIN,
			ECUType:      cleanedID,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			HWVersion:    ecu.HWVersion,
			BatterySOH:   data.SOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}

		// 채널에 담아 Bulk Insert 워커로 전달
		s.dataChan <- entity
	}
	return nil
}

// AnalyzeAndSaveBinary: 실무 규격(바이너리 오프셋) 기반 고속 파싱 및 분석
func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
	// 1. [표준 준수] DLT 패킷 해석 및 순수 페이로드 분리 (헤더 16바이트 제거)
	payloadStr := string(payload)
	vinStartIndex := strings.Index(payloadStr, vinFromTopic)

	var targetData []byte
	if vinStartIndex != -1 {
		// 앞에 "5?ICU INV " 같은 쓰레기 값이 몇 바이트든 상관없이,
		// VIN이 시작하는 지점부터 끝까지 깔끔하게 잘라냅니다.
		targetData = payload[vinStartIndex:]
	} else {
		// [안전망] 만약 원본에도 없다면 DLT 파싱을 시도해봄
		pureData, err := protocol.ParseDltPacket(payload)
		if err != nil {
			return fmt.Errorf("DLT 패킷 파싱 실패")
		}
		targetData = pureData
	}

	// 2. 잘라낸 데이터(VIN부터 시작)를 파싱
	return s.parseBody(vinFromTopic, targetData)
}

// parseBody: VIN(17) + SOH(4) + ECUs(7*N) 구조를 실제 파싱
func (s *OTAAnalyzer) parseBody(vin string, data []byte) error {
	// 최소 길이 검증: VIN(17) + SOH(4) = 21바이트
	if len(data) < 21 {
		return fmt.Errorf("데이터 길이 부족 (최소 21바이트 필요, 현재 %d)", len(data))
	}

	// 1. 배터리 SOH 추출 (17~21 오프셋)
	sohRaw := binary.BigEndian.Uint32(data[17:21])
	batterySOH := float64(math.Float32frombits(sohRaw))
	// SOH 예외 처리 (쓰레기 값 방어)
	if math.IsNaN(batterySOH) || batterySOH > 1.0 || batterySOH < 0.0 {
		batterySOH = 0.95
	}

	// 2. ECU 데이터 루프 처리 (21번 오프셋부터 7바이트씩 무한 파싱)
	foundAny := false
	for offset := 21; offset+7 <= len(data); offset += 7 {
		id := strings.TrimSpace(string(data[offset : offset+4]))

		// 하드코딩(ecuNames) 제거! 영문 알파벳 대문자로 시작하는지만 검사하여 범용성 확보
		if id == "" || id[0] < 'A' || id[0] > 'Z' {
			continue
		}

		foundAny = true
		major := int(data[offset+4])
		minor := int(data[offset+5])
		patch := int(data[offset+6])

		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

		s.dataChan <- repository.VehicleInfo{
			VIN:          vin,
			ECUType:      id,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			HWVersion:    "HW_REV_01",
			BatterySOH:   batterySOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}
	}

	if !foundAny {
		log.Printf("[Warning] VIN %s: 유효 ECU 없음. RAW: %X", vin, data)
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
