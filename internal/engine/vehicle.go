package engine

/*
 * 핵심 최적화:
 * 1. Thundering Herd 해결: 고정 주기가 아닌 랜덤 지터를 도입하여 트래픽 스파이크를 원천 차단했습니다.
 * 2. 리소스 독립성: go 키워드를 활용한 비동기 전송으로 개별 차량의 통신 지연이 전체 1,000대 시뮬레이션 성능에 영향을 주지 않도록 격리했습니다.
 * 3. 도메인 특화 최적화: Autosar DLT 표준 바이너리를 사용하여 JSON 대비 네트워크 대역폭 사용량을 획기적으로 절감했습니다.
 */

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"ota-agent/internal/transport"
	"time"

	"ota-agent/internal/collector"
	"ota-agent/internal/protocol"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Vehicle : 가상 차량 객체 정의
type Vehicle struct {
	ID     int         // 차량 고유 식별자
	Client mqtt.Client // MQTT 공유 클라이언트 주입
}

// Start : 개별 가상 차량의 데이터 수집 및 전송 루프 실행
func (v *Vehicle) Start(ctx context.Context) {
	// 차량별 독립적인 Jitter 생성을 위한 Seed 설정
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(v.ID)))
	// 프로젝트 루트 기준 데이터 경로 설정
	configPath := fmt.Sprintf("data/inventory/vehicle_%04d.json", v.ID)

	baseDelay := 10 * time.Second
	currentDelay := baseDelay
	maxDelay := 1 * time.Minute

	for {
		// 1. 차량 인벤토리 데이터 수집 (JSON 파일 로드)
		inv, err := collector.LoadInventory(configPath)
		if err != nil {
			log.Printf("[Vehicle-%04d] 수집 오류: %v. 백오프 적용.", v.ID, err)
			if currentDelay < maxDelay {
				currentDelay *= 2 // 지수 백오프 (Exponential Backoff)
			}
		} else {
			// 성공 시 대기 시간 복구 및 랜덤 지터(Jitter) 적용
			// Thundering Herd 현상 방지: ±20% 범위의 지터
			jitterRange := int64(baseDelay / 5)
			jitter := time.Duration(r.Int63n(jitterRange*2) - jitterRange)
			currentDelay = baseDelay + jitter

			// 2. [표준 규격] Autosar DLT 페이로드 조립
			var payloadBuf bytes.Buffer

			// VIN: 17바이트를 정확히 할당하여 복사 (밀림 방지)
			vinBytes := make([]byte, 17)
			for i := 0; i < 17; i++ {
				vinBytes[i] = ' '
			} // 공백 패딩
			copy(vinBytes, inv.VIN)
			payloadBuf.Write(vinBytes)

			// ECUs: 고정 폭 필드로 직렬화
			for _, ecu := range inv.ECUs {
				// ECU ID: 4바이트 고정
				ecuIDField := make([]byte, 4)
				for i := range ecuIDField {
					ecuIDField[i] = ' '
				}
				copy(ecuIDField, ecu.ID)
				payloadBuf.Write(ecuIDField)

				// SW 버전: 가변 길이를 고려하여 Null 종료 문자(\x00) 추가
				payloadBuf.WriteString(ecu.SWVersion + "\x00")
			}

			// 3. Autosar DLT 표준 패킷 직렬화
			// "ICU " (Context ID), "INV " (Message ID) 표준 식별자 사용
			binaryData, err := protocol.CreateDltPacket("ICU ", "INV ", payloadBuf.Bytes())
			if err == nil {
				// 고루틴 전송 시 데이터 오염 방지를 위해 복사본 생성
				sendData := make([]byte, len(binaryData))
				copy(sendData, binaryData)

				// 비동기 논블로킹 전송: 전송 지연이 시뮬레이션 전체에 영향을 주지 않도록 격리
				// 원본 대신 복사본(sendData)을 전달
				go transport.SendToBroker(v.Client, inv.VIN, sendData)
			} else {
				log.Printf("[Vehicle-%04d] DLT 패킷 생성 실패: %v", v.ID, err)
			}
		}

		// 4. 인터럽트 가능한 대기 (Graceful Shutdown 지원)
		select {
		case <-ctx.Done():
			log.Printf("[Vehicle-%04d] 안전하게 종료되었습니다.", v.ID)
			return
		case <-time.After(currentDelay):
		}
	}
}
