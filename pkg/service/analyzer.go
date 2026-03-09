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
	pureData, err := protocol.ParseDltPacket(payload)
	if err != nil {
		return fmt.Errorf("DLT 패킷 해석 실패: %v", err)
	}

	// 2. [범용 로직] 앞에서부터 훑으며 VIN(17자리)의 시작점을 찾음
	// WNK가 아니더라도 영문 대문자+숫자 조합의 17자리를 찾는 정규식 등을 쓸 수 있지만,
	// 가장 확실한 건 에이전트가 '데이터만' 보내게 만드는 것입니다.

	// 우선 현재 로그처럼 앞에 쓰레기 값이 붙어온다면,
	// 실제 VIN(vinFromTopic)이 바이너리 내 어디에 있는지 위치를 찾습니다.
	vinStartIndex := strings.Index(string(pureData), vinFromTopic)
	if vinStartIndex == -1 {
		// 만약 바이너리 안에 VIN이 없다면? 토픽 VIN을 쓰고 바이너리는 통째로 데이터로 간주
		log.Printf("[Info] 패킷 내 VIN 미포함. 토픽 VIN(%s) 사용", vinFromTopic)
		return s.parseBody(vinFromTopic, pureData)
	}

	// 3. VIN 위치를 찾았다면 그 이후부터 정해진 규격대로 파싱
	bodyData := pureData[vinStartIndex:]
	return s.parseBody(vinFromTopic, bodyData)
}

// parseBody: VIN(17) + SOH(4) + ECUs(7*N) 구조를 실제 파싱
func (s *OTAAnalyzer) parseBody(vin string, data []byte) error {
	// 최소 길이 검증: VIN(17) + SOH(4) = 21바이트
	if len(data) < 21 {
		return fmt.Errorf("데이터 길이 부족 (최소 21바이트 필요, 현재 %d)", len(data))
	}

	// 1. 배터리 SOH 추출 (VIN 17바이트 직후 4바이트)
	// IEEE 754 float32 값을 읽어옵니다.
	sohRaw := binary.BigEndian.Uint32(data[17:21])
	batterySOH := float64(math.Float32frombits(sohRaw))

	// 2. ECU 데이터 루프 처리 (21번 오프셋부터 7바이트씩 절단)
	foundAny := false
	for offset := 21; offset+7 <= len(data); offset += 7 {
		// ECU ID (4바이트, 예: "ADAS", "BMS ")
		id := strings.TrimSpace(string(data[offset : offset+4]))
		if id == "" || id[0] < 'A' || id[0] > 'Z' {
			continue // 유효하지 않은 ECU ID 건너뛰기
		}

		foundAny = true
		// 버전 정보 (3바이트: Major, Minor, Patch)
		major := int(data[offset+4])
		minor := int(data[offset+5])
		patch := int(data[offset+6])

		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

		// 3. DB 적재 채널로 전송
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
		log.Printf("[Warning] VIN %s: 유효한 ECU 정보를 찾지 못했습니다.", vin)
	} else {
		// 디버깅용: 적재 시도 로그 (실제 운영 시에는 제거하거나 레벨 조정)
		// log.Printf("[Success] VIN %s 데이터 분석 완료 및 채널 전송", vin)
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
