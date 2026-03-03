package engine

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
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
		// transport 패키지의 도구를 호출하여 에러 보고
		transport.SendStatus(v.Client, v.VIN, "ERR_HASH_MISMATCH")
		return
	}

	// 성공 시 보고
	transport.SendStatus(v.Client, v.VIN, "SUCCESS_VERIFIED")
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
	// 1. 고정 크기 버퍼 할당: 런타임 오버헤드 최소화
	// 구조: VIN(17) + SOH(4) + (ECU_ID(4) + Ver(3)) * N
	buf := make([]byte, 128)

	// 2. VIN 직렬화 (0~16 index): 고정 17바이트
	copy(buf[0:17], fmt.Sprintf("%-17s", inv.VIN))

	// 3. SOH 직렬화 (17~20 index): BigEndian IEEE 754 float32
	binary.BigEndian.PutUint32(buf[17:21], math.Float32bits(float32(inv.SOH)))

	// 4. ECU 데이터 직렬화 (21 index ~)
	offset := 21
	for _, ecu := range inv.ECUs {
		// ECU ID: 4바이트 고정 (예: "BMS ")
		copy(buf[offset:offset+4], fmt.Sprintf("%-4s", ecu.ID))

		// [고도화] 버전 문자열(v2.3.5) -> 3바이트 정수 바이너리 변환
		// 텍스트 "2.3.5"(5바이트) 대비 약 40% 크기 절감
		major, minor, patch := parseVersionToInt(ecu.SWVersion)
		buf[offset+4] = byte(major)
		buf[offset+5] = byte(minor)
		buf[offset+6] = byte(patch)

		offset += 7
		if offset+7 > len(buf) {
			break
		} // 버퍼 오버플로우 방지
	}

	// 5. DLT 패킷 생성 및 비동기 전송
	binaryData, err := protocol.CreateDltPacket("ICU ", "INV ", buf[:offset])
	if err == nil {
		// 전송용 슬라이스 복사 (고루틴 안전성 확보)
		sendData := make([]byte, len(binaryData))
		copy(sendData, binaryData)
		go transport.SendToBroker(v.Client, v.VIN, sendData)
	}

	fmt.Printf("[%s] [Optimization-Log] Binary Payload Size: %d bytes (JSON 대비 %d%% 절감)\n",
		v.VIN, len(binaryData), 100-(len(binaryData)*100/210))
}

// [헬퍼] 버전 문자열을 정수형 바이트로 파싱
func parseVersionToInt(vStr string) (int, int, int) {
	vStr = strings.TrimPrefix(vStr, "v")
	parts := strings.Split(vStr, ".")
	if len(parts) < 3 {
		return 0, 0, 0
	}
	major, _ := strconv.Atoi(parts[0])
	minor, _ := strconv.Atoi(parts[1])
	patch, _ := strconv.Atoi(parts[2])
	return major, minor, patch
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
