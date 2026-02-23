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
	// K8s Deployment의 env 설정에서 주입될 값
	vin := os.Getenv("VEHICLE_VIN")
	if vin == "" {
		// 로컬 테스트용 기본값 (K8s 없이 단독 실행 시 사용)
		vin = "VIN-DEBUG-0001"
	}

	// 2. 브로커 주소 환경 변수화
	// 로컬에서는 localhost지만, K8s 내부에서는 서비스 이름(예: mqtt-svc)으로 접근합니다.
	broker := os.Getenv("MQTT_BROKER_URL")
	if broker == "" {
		broker = "tcp://localhost:1885"
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
