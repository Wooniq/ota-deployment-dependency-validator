package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	// 1. 실제 비즈니스 로직이 들어있는 engine 패키지 임포트
	"github.com/Wooniq/ota-agent/internal/engine"

	// 2. MQTT 타입을 사용하기 위한 외부 패키지 임포트
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	const totalVehicles = 1000
	// 1. 시스템 종료 신호를 감지하기 위한 Context 생성 (Graceful Shutdown)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// [ADR 0001] MQTT 공유 클라이언트 초기화
	opts := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883")
	client := mqtt.NewClient(opts)

	var wg sync.WaitGroup
	log.Printf("%d 대의 가상 차량 시뮬레이션 시작 (exit : Ctrl + C)", totalVehicles)

	for i := 1; i <= totalVehicles; i++ {
		wg.Add(1)

		// Startup Staggering: 초기 부하 분산
		time.Sleep(20 * time.Millisecond)

		go func(id int) {
			defer wg.Done()

			// engine 패키지의 Vehicle 구조체 활용
			v := engine.Vehicle{
				ID:     id,
				Client: client,
			}

			v.Start(ctx)
		}(i)

		// 1,000대가 동시에 소켓을 여는 '접속 폭주'를 막기 위한 간격 추가
		time.Sleep(20 * time.Millisecond)
	}

	wg.Wait()
	log.Println("모든 에이전트 종료")
}
