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
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// OTAAnalyzer: 차량 데이터 분석 및 SAP HANA DB 적재를 담당하는 서비스 객체
type OTAAnalyzer struct {
	Repo       *repository.HANARepository
	TargetADAS string
	TargetBMS  string
	dataChan   chan repository.VehicleInfo // 비동기 DB 적재를 위한 데이터 채널
	batchSize  int                         // 벌크 Insert 단위
	MQTTClient mqtt.Client                 // 롤백 명령을 보내기 위한 MQTT 클라이언트
}

// NewOTAAnalyzer: 분석기 인스턴스를 생성하고 백그라운드 배치 워커를 가동함
func NewOTAAnalyzer(repo *repository.HANARepository, mqttClient mqtt.Client, adasVer, bmsVer string) *OTAAnalyzer {
	a := &OTAAnalyzer{
		Repo:       repo,
		TargetADAS: adasVer,
		TargetBMS:  bmsVer,
		dataChan:   make(chan repository.VehicleInfo, 2000),
		batchSize:  100,
		MQTTClient: mqttClient, // 주입받은 MQTT 클라이언트 저장
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
		cleanedID := strings.TrimSpace(ecu.ID)

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
		targetData = payload[vinStartIndex:]
	} else {
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

		if id == "" || id[0] < 'A' || id[0] > 'Z' {
			continue
		}

		foundAny = true
		major := int(data[offset+4])
		minor := int(data[offset+5])
		patch := int(data[offset+6])

		versionStr := fmt.Sprintf("%d.%d.%d", major, minor, patch)
		// 1. 상태 분석 (여기서 status가 결정됨)
		status, needsUpdate := s.performDeepAnalysis(versionStr, versionStr, batterySOH)

		// 2. [Trigger] 상태 기반 롤백 명령
		// performDeepAnalysis에서 결정된 status가 'StatusBatteryLow'라면 롤백
		if status == repository.StatusBatteryLow {
			s.TriggerRollback(vin, id)
		}

		// 3. DB 적재용 채널 전송
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

// TriggerRollback: 에러 발생 차량에게 이전 안정 버전으로 복구하라는 명령 하달
// 대규모 Mass Request 상황에서 Kafka Consumer 병목을 막기 위해 고루틴(비동기)으로 실행
func (s *OTAAnalyzer) TriggerRollback(vin string, targetECU string) {
	if s.MQTTClient == nil {
		log.Printf("[Rollback-Error] MQTT Client가 초기화되지 않아 명령을 보낼 수 없습니다.")
		return
	}

	// 비동기 스레드 분리
	go func(v, ecu string) {
		log.Printf("[Rollback-Init] VIN: %s의 %s 제어기 복구 절차 시작", v, ecu)

		// HANA DB에서 '직전 성공 버전(Last Known Good Configuration)' 동적 조회
		stableVer, stablePath, stableHash, err := s.Repo.GetLastStableFirmware(v, ecu)
		if err != nil {
			log.Printf("[Rollback-Warning] VIN: %s (%s)의 이전 정상 버전을 찾을 수 없습니다. 공장 초기화 버전으로 대체합니다. (사유: %v)", v, ecu, err)
			// DB 조회 실패 시 최후의 보루(Fallback) 설정
			stableVer = "v1.0.0"
			stablePath = fmt.Sprintf("/firmware/%s/factory_default.bin", ecu)
			stableHash = "FACTORY_DEFAULT_SAFE_HASH"
		}

		// 2. 에이전트 규격에 맞는 페이로드 조립
		cmdPayload := map[string]string{
			"action":        "rollback",
			"file_path":     stablePath,
			"expected_hash": stableHash,
			"version":       stableVer,
		}

		data, err := json.Marshal(cmdPayload)
		if err != nil {
			log.Printf("[Rollback-Error] 페이로드 조립 실패: %v", err)
			return
		}

		// 3. 해당 차량의 Command 토픽으로 Publish
		topic := fmt.Sprintf("ota/command/%s", v)
		token := s.MQTTClient.Publish(topic, 1, false, data)
		token.Wait() // 고루틴 내부이므로 Wait() 해도 메인 Kafka 루프에 영향 없음

		if token.Error() != nil {
			log.Printf("[Rollback-Fatal] VIN: %s 롤백 명령 발송 실패! (%v)", v, token.Error())

			// 실패 이력도 DB에 남김
			s.Repo.RecordUpdateHistory(v, ecu, "ERR_ROLLBACK_PUBLISH_FAILED", stableVer)
		} else {
			log.Printf("[Anti-Bricking] VIN: %s (%s) 해시 불일치 감지! 이전 정상 버전(%s) 복구 명령 발송 완료!", v, ecu, stableVer)

			// [제언 2 반영] 상태 추적 및 이력 관리 (Audit Trail)
			// 나중에 관제 대시보드에서 '롤백 조치된 차량 목록'을 띄우기 위한 기록
			err = s.Repo.RecordUpdateHistory(v, ecu, "ERR_ROLLBACK_INITIATED", stableVer)
			if err != nil {
				log.Printf("[Audit-Warning] VIN: %s 롤백 이력 DB 기록 실패: %v", v, err)
			}
		}
	}(vin, targetECU) // 클로저 변수 캡처 문제 방지를 위해 파라미터로 넘김
}
