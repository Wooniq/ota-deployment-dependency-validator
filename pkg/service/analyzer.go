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
/*func (s *OTAAnalyzer) AnalyzeAndSave(vin, payload string) error {
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
}*/

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
	pureData, err := protocol.ParseDltPacket(payload)
	if err != nil {
		return fmt.Errorf("DLT 패킷 해석 실패: %v", err)
	}

	// 2. [규격 검증] 최소 길이 확인 (VIN 17바이트 + SOH 4바이트 = 최소 21바이트)
	dataLen := len(pureData)
	if dataLen < 21 {
		return fmt.Errorf("부적절한 패킷 길이: %d (최소 21 필요)", dataLen)
	}

	// 3. VIN 추출 (0~17 오프셋)
	// 에이전트에서 "%-17s"로 넣은 공백을 TrimSpace로 깔끔하게 제거합니다.
	vin := strings.TrimSpace(string(pureData[0:17]))
	if !strings.HasPrefix(vin, "WNK") {
		log.Printf("[Security-Alert] 비정상 VIN 패턴 감지: %s, 토픽 정보(%s)로 보정", vin, vinFromTopic)
		vin = vinFromTopic
	}

	// 4. 배터리 SOH 추출 (17~21 오프셋)
	sohRaw := binary.BigEndian.Uint32(pureData[17:21])
	batterySOH := float64(math.Float32frombits(sohRaw))

	// 5. ECU 데이터 루프 처리 (21 오프셋부터 7바이트씩)
	foundAny := false
	for offset := 21; offset+7 <= dataLen; offset += 7 {
		// ECU ID 추출 (4바이트)
		id := strings.TrimSpace(string(pureData[offset : offset+4]))
		if id == "" {
			continue
		}

		foundAny = true
		// 버전 정보 추출 (3바이트: Major, Minor, Patch)
		major := int(pureData[offset+4])
		minor := int(pureData[offset+5])
		patch := int(pureData[offset+6])

		// 1. 추출한 숫자를 다시 "2.3.5" 형태의 문자열로 복원
		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)

		// 2. 분석 엔진 호출
		status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

		// 6. [비동기 처리] 파이프라인 전송 (Mass Request 대응을 위한 채널 활용)
		s.dataChan <- repository.VehicleInfo{
			VIN:          vin,
			ECUType:      id,
			SWMajor:      major,
			SWMinor:      minor,
			SWPatch:      patch,
			HWVersion:    "HW_REV_01", // 실무에선 이 또한 바이너리에서 추출
			BatterySOH:   batterySOH,
			UpdateStatus: status,
			LastReported: time.Now(),
			NeedsUpdate:  needsUpdate,
		}
	}

	if !foundAny {
		log.Printf("[Warning] VIN %s: 유효한 ECU 인벤토리 정보 없음", vin)
	}

	return nil
}

/*func (s *OTAAnalyzer) AnalyzeAndSaveBinary(vinFromTopic string, payload []byte) error {
    // 1. DLT 패킷 해석
    pureData, err := protocol.ParseDltPacket(payload)
    if err != nil {
       return fmt.Errorf("DLT 해석 실패: %v", err)
    }

    // [현업 규격] 최소 VIN(17)은 반드시 있어야 함
    dataLen := len(pureData)
    if dataLen < 17 {
       return fmt.Errorf("부정확한 패킷 길이: %d (최소 17 필요)", dataLen)
    }

    // 2. VIN 추출 및 유효성 검사
    vin := strings.TrimSpace(string(pureData[0:17]))
    if len(vin) < 10 { // 기본적인 VIN 형식 검증
        return fmt.Errorf("유효하지 않은 VIN: %s", vin)
    }

    // 3. SOH 추출 (데이터가 충분할 때만)
    var batterySOH float64 = 0.0
    if dataLen >= 21 {
       sohRaw := binary.BigEndian.Uint32(pureData[17:21])
       batterySOH = float64(math.Float32frombits(sohRaw))
    }

    // 4. 가변 ECU 데이터 루프 처리
    // 시작 오프셋: 21 (VIN 17 + SOH 4)
    // 한 세트: 7바이트 (ID 4 + Ver 3)
    foundECU := false
    for offset := 21; offset+7 <= dataLen; offset += 7 {
       ecuID := strings.TrimSpace(string(pureData[offset : offset+4]))
       if ecuID == "" { continue }

       foundECU = true
       major, minor, patch := int(pureData[offset+4]), int(pureData[offset+5]), int(pureData[offset+6])
       versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)

       status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

       // 5. DB 엔티티 생성 및 채널 전송
       s.dataChan <- repository.VehicleInfo{
          VIN:          vin,
          ECUType:      ecuID,
          SWMajor:      major,
          SWMinor:      minor,
          SWPatch:      patch,
          HWVersion:    "HW_REV_01", // 실제 현업에선 이 또한 패킷에서 추출함
          BatterySOH:   batterySOH,
          UpdateStatus: status,
          LastReported: time.Now(),
          NeedsUpdate:  needsUpdate,
       }
    }

    // ECU 정보가 없는 패킷일 경우 VIN과 SOH라도 기본 적재 (옵션)
    if !foundECU {
        log.Printf("[Info] VIN %s: ECU 정보 없음, 기본 상태만 기록", vin)
    }

    return nil
}*/

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
