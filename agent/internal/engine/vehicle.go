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
	"time"

	"github.com/Wooniq/ota-agent/internal/collector"
	"github.com/Wooniq/ota-agent/internal/protocol"
	"github.com/Wooniq/ota-agent/internal/transport"
	
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Vehicle : 가상 차량 객체 정의
type Vehicle struct {
	ID     int         // 차량 고유 식별자
	Client mqtt.Client // MQTT 공유 클라이언트 주입
}

// Start : 개별 가상 차량의 데이터 수집 및 전송 루프 실행
func (v *Vehicle) Start(ctx context.Context) {
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(v.ID)))     // 차량별 독립적인 Jitter 생성을 위한 Seed 설정
	configPath := fmt.Sprintf("../data/inventory/vehicle_%04d.json", v.ID) // 상대 경로: agent 폴더에서 실행 기준

	baseDelay := 10 * time.Second
	currentDelay := baseDelay
	maxDelay := 1 * time.Minute

	for {
		// 1. 차량 인벤토리 데이터 수집
		inv, err := collector.LoadInventory(configPath)
		if err != nil {
			log.Printf("[Vehicle-%04d] 수집 오류: %v. 백오프 적용.", v.ID, err)
			if currentDelay < maxDelay {
				currentDelay *= 2 // 지수 백오프
			}
		} else {
			// 1. 성공 시 대기 시간 복구 및 랜덤 지터(Jitter) 적용
			// 모든 차량이 동시에 데이터를 쏘는 Thundering Herd 현상 방지
			jitterRange := int64(baseDelay / 5) // ±20% (약 2초) 범위의 지터
			jitter := time.Duration(r.Int63n(jitterRange*2) - jitterRange)
			currentDelay = baseDelay + jitter

			// 2. DLT 표준 바이너리 페이로드 조립
			var payloadBuf bytes.Buffer
			payloadBuf.WriteString(inv.VIN)
			for _, ecu := range inv.ECUs {
				payloadBuf.WriteString(ecu.ID)        // 4바이트 ECU ID
				payloadBuf.WriteString(ecu.SWVersion) // 가변 SW 버전
			}

			// 3. Autosar DLT 표준 패킷 직렬화 (전송 효율 극대화)
			binaryData, err := protocol.CreateDltPacket("ICU ", "INV ", payloadBuf.Bytes())
			if err == nil {
				// 비동기 논블로킹 전송
				// 전송 지연이 발생해도 시뮬레이션 루프가 멈추지 않도록 'go' 키워드 사용
				go transport.SendToBroker(v.Client, inv.VIN, binaryData)
			}
		}

		// 4. 인터럽트 가능한 대기
		select {
		case <-ctx.Done(): // 상위 매니저의 종료 신호(Graceful Shutdown) 수신
			log.Printf("[Vehicle-%04d] 안전하게 종료되었습니다.", v.ID)
			return
		case <-time.After(currentDelay):
		}
	}
}
