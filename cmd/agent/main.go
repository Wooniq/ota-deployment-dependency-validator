package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"syscall"

	"github.com/Wooniq/ota-agent/pkg/engine"
	"github.com/Wooniq/ota-agent/pkg/transport"
)

func main() {
	// 1. 환경 변수로부터 차량 식별 정보(VIN) 로드
	// YAML의 MY_POD_NAME(ota-agent-0, 1...)을 VIN으로 사용합니다.
	vin := os.Getenv("MY_POD_NAME")
	if vin == "" {
		vin = "VIN-DEBUG-0001"
	}

	// 2. 브로커 주소 환경 변수화
	broker := os.Getenv("BROKER_URL")
	if broker == "" {
		broker = "tcp://localhost:1883"
	}

	// 3. Graceful Shutdown을 위한 컨텍스트 설정
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 4. 이 차량 전용 MQTT 클라이언트 생성 및 연결
	// transport.NewMQTTClient 내부에서 ClientID를 VIN으로 설정하도록 되어있어야 합니다.
	client, err := transport.NewMQTTClient(broker, vin)
	if err != nil {
		log.Fatalf("[%s] MQTT 브로커 연결 실패: %v", vin, err)
	}
	log.Printf("[%s] MQTT 연결 성공: %s", vin, broker)

	// 5. engine.Vehicle 구조체 초기화
	v := engine.Vehicle{
		VIN:    vin,
		Client: client,
	}

	log.Printf("=== OTA Agent 시작 [VIN: %s] ===", v.VIN)

	// 6. 에이전트 실행 (Start 내부에서 OnUpdateReceived와 루프가 동작함)
	v.Start(ctx)

	log.Printf("[%s] 에이전트 종료 프로세스 완료", vin)
}
