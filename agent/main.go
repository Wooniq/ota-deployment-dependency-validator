package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Wooniq/ota-agent/internal/engine"
	"github.com/Wooniq/ota-agent/internal/transport"

	// 2. MQTT 타입을 사용하기 위한 외부 패키지 임포트
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	const totalVehicles = 1000
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. MQTT 옵션 설정
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://localhost:1885")
	opts.SetClientID("ota-main-collector") // 메인 클라이언트 ID
	opts.SetAutoReconnect(true)

	// 2. 클라이언트 생성 및 실제 연결
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT 브로커 연결 실패: %v", token.Error())
	}
	log.Println("MQTT 브로커 연결 성공 (port: 1885)")

	var wg sync.WaitGroup
	log.Printf("%d 대의 가상 차량 시뮬레이션 시작 (exit : Ctrl + C)", totalVehicles)

	for i := 1; i <= totalVehicles; i++ {
		wg.Add(1)
		vin := fmt.Sprintf("VIN-%06d", i)

		go func(id int, v_vin string) {
			defer wg.Done()

			// transport 패키지의 함수를 사용하여 각자 연결
			client, err := transport.NewMQTTClient("tcp://localhost:1885", v_vin)
			if err != nil {
				log.Printf("VIN:%s 연결 실패: %v", v_vin, err)
				return
			}

			v := engine.Vehicle{
				ID:     id,
				Client: client,
			}
			v.Start(ctx)
		}(i, vin)

		time.Sleep(20 * time.Millisecond)
	}

	// 프로그램이 바로 종료되지 않도록 대기
	<-ctx.Done()
	log.Println("종료 신호 감지, 시뮬레이션을 중단합니다...")

	// 모든 고루틴이 정리될 때까지 잠시 대기
	client.Disconnect(250)
	log.Println("모든 에이전트 종료")
}
