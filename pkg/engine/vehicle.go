package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/Wooniq/ota-agent/pkg/collector"
	"github.com/Wooniq/ota-agent/pkg/protocol"
	"github.com/Wooniq/ota-agent/pkg/transport"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Vehicle struct {
	VIN    string
	Client mqtt.Client
}

// 1. SHA-256 무결성 검증 (ISO 21434 보안 규격 준수 시뮬레이션)
func (v *Vehicle) VerifyFirmware(filePath string, expectedHash string) (bool, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return false, err
	}
	defer file.Close()

	hash := sha256.New()
	// io.Copy를 사용하여 메모리 효율적 스트리밍 해시 계산 (성능 최적화)
	if _, err := io.Copy(hash, file); err != nil {
		return false, err
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))
	return actualHash == expectedHash, nil
}

// 2. 업데이트 이벤트 핸들러
func (v *Vehicle) OnUpdateReceived(filePath string, expectedHash string) {
	log.Printf("[%s] OTA 패키지 수신: %s", v.VIN, filePath)

	// 무결성 검증 수행
	success, err := v.VerifyFirmware(filePath, expectedHash)
	if err != nil || !success {
		log.Printf("[%s] 검증 실패: %v", v.VIN, err)
		v.reportStatus("ERR_HASH_MISMATCH") // 에러 코드 체계화
		return
	}

	log.Printf("[%s] 무결성 검증 통과 (SHA-256 일치)", v.VIN)
	v.reportStatus("SUCCESS_VERIFIED")
}

// 3. 업데이트 명령 구독 설정
func (v *Vehicle) setupUpdateSubscriber() {
	topic := fmt.Sprintf("ota/command/%s", v.VIN)
	v.Client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		var cmd struct {
			FilePath     string `json:"file_path"`
			ExpectedHash string `json:"expected_hash"`
		}
		if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
			log.Printf("[%s] Payload Parsing Error: %v", v.VIN, err)
			return
		}
		// 논블로킹 실행: 검증 중에도 상태 보고 루프가 멈추지 않도록 격리
		go v.OnUpdateReceived(cmd.FilePath, cmd.ExpectedHash)
	})
	log.Printf("[%s] MQTT 구독 활성화: %s", v.VIN, topic)
}

// 4. Start: 에이전트 메인 루프 (1대 독립 실행형)
func (v *Vehicle) Start(ctx context.Context) {
	// 업데이트 명령 대기 시작
	v.setupUpdateSubscriber()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	baseDelay := 10 * time.Second

	// K8s 환경에서 각 Pod에 마운트된 고유 인벤토리 경로
	configPath := "/etc/ota/inventory.json"

	for {
		// 인벤토리 수집 및 DLT 패킷 전송
		inv, err := collector.LoadInventory(configPath)
		if err == nil {
			v.sendDltInventory(inv)
		} else {
			log.Printf("[%s] Inventory Load Error: %v", v.VIN, err)
		}

		// Thundering Herd 방지를 위한 랜덤 지터 적용 대기
		select {
		case <-ctx.Done():
			return
		case <-time.After(v.getNextJitterDelay(r, baseDelay)):
		}
	}
}

// 5. [내부 헬퍼] DLT 바이너리 조립 및 전송 로직
func (v *Vehicle) sendDltInventory(inv *collector.VehicleInventory) {
	var payloadBuf bytes.Buffer

	// VIN: 17바이트 고정 폭 직렬화 (공백 패딩)
	vinBytes := make([]byte, 17)
	for i := range vinBytes {
		vinBytes[i] = ' '
	}
	copy(vinBytes, inv.VIN)
	payloadBuf.Write(vinBytes)

	// ECUs: 고정 폭 필드로 직렬화 (Autosar 표준 모사)
	for _, ecu := range inv.ECUs {
		// ECU ID: 4바이트 고정 (예: BMS , ADAS)
		ecuIDField := make([]byte, 4)
		for i := range ecuIDField {
			ecuIDField[i] = ' '
		}
		copy(ecuIDField, ecu.ID)
		payloadBuf.Write(ecuIDField)

		// SW 버전: 가변 길이를 고려하여 Null 종료 문자(\x00) 추가
		payloadBuf.WriteString(ecu.SWVersion + "\x00")
	}

	// Autosar DLT 표준 패킷 생성 ("ICU " 컨텍스트 사용)
	binaryData, err := protocol.CreateDltPacket("ICU ", "INV ", payloadBuf.Bytes())
	if err == nil {
		// 비동기 전송으로 메인 루프 지연 방지
		sendData := make([]byte, len(binaryData))
		copy(sendData, binaryData)
		go transport.SendToBroker(v.Client, v.VIN, sendData)
	}
}

func (v *Vehicle) getNextJitterDelay(r *rand.Rand, base time.Duration) time.Duration {
	jitterRange := int64(base / 5)
	jitter := time.Duration(r.Int63n(jitterRange*2) - jitterRange)
	return base + jitter
}

func (v *Vehicle) reportStatus(status string) {
	topic := fmt.Sprintf("ota/status/%s", v.VIN)
	payload := map[string]string{
		"vin":       v.VIN,
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	data, _ := json.Marshal(payload)
	v.Client.Publish(topic, 1, false, data)
}
