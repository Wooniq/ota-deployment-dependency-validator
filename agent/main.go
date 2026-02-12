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

	"github.com/Wooniq/ota-agent/internal/collector"
	"github.com/Wooniq/ota-agent/internal/protocol"
)

func main() {
	const totalVehicles = 1000
	// 1. 시스템 종료 신호를 감지하기 위한 Context 생성 (Graceful Shutdown)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	log.Printf("%d 대의 가상 차량 시뮬레이션 시작 (exit : Ctrl + C)", totalVehicles)

	for i := 1; i <= totalVehicles; i++ {
		wg.Add(1)
		go func(vehicleId int) {
			defer wg.Done()

			configPath := fmt.Sprintf("agent/config/vehicle_%04d.json", vehicleID)
			baseDelay := 10 * time.Second
			currentDelay := baseDelay
			maxDelay := 1 * time.Minute // 최대 백오프 시간

			for {
				// 1. 데이터 수집 및 패킷 생성 로직
				inv, err := collector.LoadInventory(configPath)
				if err != nil {
					log.Printf("[Vehicle-%04d] 수집 오류: %v. 백오프 적용.", vehicleID, err)

					// 지수 백오프 적용 : 오류 시 대기 시간 2배 증가
					if currentDelay < maxDelay {
						currentDelay *= 2
					}
				} else {
					// 성공 시 대기 시간 초기화
					currentDelay = baseDelay

					payload := []byte(fmt.Sprintf("VIN:%s-TIME:%d", inv.VIN, time.Now().Unix()))
					binaryData, err := protocol.CreateDltPacket("ICU ", "INV ", payload)

					if err == nil {
						// 2. 메시지 전송 추상화 (Transport)
						// TODO: http.Post 또는 Kafka Producer 연결
						sendToBackend(vehicleID, binaryData)
					}
				}

				// 3. 인터럽트 가능한 대기 (Interruptible Sleep)
				select {
				case <-ctx.Done():
					log.Printf("[Vehicle-%04d] 전송을 중단하고 안전하게 종료", vehicleID)
					return
				case <-time.After(currentDelay):
					// 정해진 시간 대기 후 루프 재시작
				}
			}
		}(i)
	}

	wg.Wait()
	log.Println("모든 에이전트 종료")
}

// sendToBackend: 전송부 로직 추상화 (End-to-End 파이프라인 확장 포인트)
func sendToBackend(id int, data []byte) {
	// 현재는 시뮬레이션을 위해 로그로 대체
	// 추후 이 부분에 전송 로직이 들어감
	log.Printf("[Transport] Vehicle-%04d -> Backend 전송 완료 (%d bytes)", id, len(data))
}
