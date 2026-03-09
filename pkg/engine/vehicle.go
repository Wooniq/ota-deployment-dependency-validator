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
	"path/filepath"
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
func (v *Vehicle) OnUpdateReceived(vin string, filePath string, expectedHash string) {
	log.Printf("[%s] OTA 패키지 수신: %s", vin, filePath)

	// 무결성 검증 수행
	success, err := v.VerifyFirmware(filePath, expectedHash)

	if err != nil || !success {
		// transport 패키지의 도구를 호출하여 에러 보고
		transport.SendStatus(v.Client, vin, "ERR_HASH_MISMATCH")
		return
	}

	// 성공 시 보고
	transport.SendStatus(v.Client, vin, "SUCCESS_VERIFIED")
}

// 4. Start: 1,000대 동시 접속 시뮬레이션 모드로 수정
func (v *Vehicle) Start(ctx context.Context) {
	inventoryDir := "data/inventory"

	// 1. 폴더 내 모든 파일 읽기
	files, err := os.ReadDir(inventoryDir)
	if err != nil {
		log.Fatalf("Inventory 폴더를 읽을 수 없습니다: %v", err)
	}

	log.Printf("=== [Massive Connection Mode] %d대 차량 동시 접속 시작 ===", len(files))

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			configPath := filepath.Join(inventoryDir, file.Name())

			// 2. 각 차량별로 새로운 고루틴 실행 (폭주 시뮬레이션)
			go func(path string) {
				// 각 가상 차량의 고유 클라이언트 생성 및 구동 로직
				// (실제 대규모 테스트 시에는 각 차량 객체가 별도의 MQTT Client를 가져야 함)
				v.runVirtualVehicle(ctx, path, r)
			}(configPath)

			// 3. 접속 폭주 시 브로커 부하를 조절하기 위한 미세 지연 (Thundering Herd 방지)
			time.Sleep(5 * time.Millisecond)
		}
	}

	// 메인 루프 유지
	<-ctx.Done()
}

// 개별 차량의 독립 실행 루틴
func (v *Vehicle) runVirtualVehicle(ctx context.Context, configPath string, r *rand.Rand) {
	baseDelay := 30 * time.Second // 보고 주기 (1,000대이므로 조금 늘림)

	// 파일에서 VIN 로드 (각 차량의 정체성 확인)
	inv, err := collector.LoadInventory(configPath)
	if err != nil {
		return
	}

	currentVIN := inv.VIN
	log.Printf("[%s] 가상 차량 접속 완료", currentVIN)

	// 이 차량의 명령 구독 설정
	// 주의: v.VIN을 currentVIN으로 동적 처리하도록 코드 수정 필요
	v.setupSpecificUpdateSubscriber(currentVIN)

	for {
		// 인벤토리 수집 및 전송
		inv, err := collector.LoadInventory(configPath)
		if err == nil {
			v.sendDltInventory(inv) // 내부에서 binary 전송
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(v.getNextJitterDelay(r, baseDelay)):
		}
	}
}

// 각 가상 차량의 고유 VIN으로 명령 구독
func (v *Vehicle) setupSpecificUpdateSubscriber(vin string) {
	topic := fmt.Sprintf("ota/command/%s", vin)

	v.Client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
		var cmd struct {
			FilePath     string `json:"file_path"`
			ExpectedHash string `json:"expected_hash"`
		}
		if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
			log.Printf("[%s] Payload Parsing Error: %v", vin, err)
			return
		}
		// 개별 VIN 컨텍스트를 유지하며 업데이트 핸들러 실행
		go v.OnUpdateReceived(vin, cmd.FilePath, cmd.ExpectedHash)
	})

	log.Printf("[%s] MQTT 구독 활성화: %s", vin, topic)
}

// 5. [내부 헬퍼] DLT 바이너리 조립 및 전송 로직
func (v *Vehicle) sendDltInventory(inv *collector.VehicleInventory) {
	// 1. 필요한 정확한 페이로드 크기 계산
	// VIN(17) + SOH(4) + (ECU 개수 * 7)
	// * ECU 세트당 7바이트: ID(4) + Major(1) + Minor(1) + Patch(1)
	payloadSize := 17 + 4 + (len(inv.ECUs) * 7)

	buf := make([]byte, payloadSize)

	// 2. VIN 직렬화 (0~17 index)
	// 17자보다 짧으면 공백으로 채우고, 길면 자름 (고정 오프셋 보장)
	vinBytes := []byte(fmt.Sprintf("%-17s", inv.VIN))
	if len(vinBytes) > 17 {
		vinBytes = vinBytes[:17]
	}
	copy(buf[0:17], vinBytes) // copy 함수: 가공된 vinBytes를 실제 전송용 버퍼(buf)에 밀어넣음

	// 3. SOH 직렬화 (17~20 index): BigEndian IEEE 754 float32
	binary.BigEndian.PutUint32(buf[17:21], math.Float32bits(float32(inv.SOH)))

	// 4. ECU 데이터 직렬화 (21 index ~)
	offset := 21
	for _, ecu := range inv.ECUs {
		// ECU ID (4바이트 고정, 예: "BMS ")
		ecuID := []byte(fmt.Sprintf("%-4s", ecu.ID))
		if len(ecuID) > 4 {
			ecuID = ecuID[:4]
		}
		copy(buf[offset:offset+4], ecuID)

		// 버전 정보 (3바이트 고정)
		major, minor, patch := parseVersionToInt(ecu.SWVersion)
		buf[offset+4] = byte(major)
		buf[offset+5] = byte(minor)
		buf[offset+6] = byte(patch)

		offset += 7
	}

	// 5. DLT 패킷 생성 및 비동기 전송
	binaryData, err := protocol.CreateDltPacket("ICU ", "INV ", buf[:offset])
	if err == nil {
		// 전송용 슬라이스 복사 (고루틴 안전성 확보)
		sendData := make([]byte, len(binaryData))
		copy(sendData, binaryData)
		go transport.SendToBroker(v.Client, inv.VIN, sendData)
	}

	// 최적화 로그: 이제 Binary Payload Size가 ECU 개수에 따라 가변적이지만 정확하게 출력됨
	fmt.Printf("[%s] [Optimization-Log] Binary Payload Size: %d bytes\n",
		inv.VIN, len(binaryData))
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

func (v *Vehicle) reportStatus(vin string, status string) {
	topic := fmt.Sprintf("ota/status/%s", vin)
	payload := map[string]string{
		"vin":       vin,
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	data, _ := json.Marshal(payload)

	// 비동기 전송 (1,000대 폭주 시 병목 방지)
	go v.Client.Publish(topic, 1, false, data)
}
