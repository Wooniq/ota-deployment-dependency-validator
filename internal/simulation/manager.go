package simulation

/*
 * 핵심 최적화:
 * 1. Startup Staggering: 1,000대 에이전트의 동시 접속으로 인한 TCP 핸드셰이크 폭주를 방지하기 위해 각 고루틴 시작 사이에 미세 지연을 두어 브로커 부하를 분산했습니다.
 * 2. 리소스 최적화 (Dependency Injection): 단일 MQTT 클라이언트를 모든 차량 객체에 주입하여 공유함으로써, 호스트 OS의 포트 고갈(Port Exhaustion)을 방지하고 시스템 리소스 점유를 최소화했습니다.
 * 3. Graceful Shutdown: Context 전파와 sync.WaitGroup을 활용하여 시스템 종료 시 모든 고루틴의 안전한 회수를 보장하고 종료 시점의 데이터 유실을 원천 차단했습니다.
 */

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/Wooniq/ota-agent/internal/engine"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// RunSimulation : 대규모 고루틴 시뮬레이션 제어
func RunSimulation(ctx context.Context, total int, client mqtt.Client) {
	var wg sync.WaitGroup

	// 루프 밖에서 랜덤 시드를 한 번만 설정하거나 전역 시드 사용 (Go 1.20 이상은 생략 가능)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 1; i <= total; i++ {
		wg.Add(1)

		// Startup Staggering: 초기 브로커 연결 부하 분산
		// 모든 차량이 0.001초 만에 동시 접속하는 것을 막아 브로커의 TCP 핸드셰이크 폭주를 방지합니다.
		time.Sleep(time.Duration(10+r.Intn(40)) * time.Millisecond)

		go func(id int) {
			defer wg.Done()

			// [관심사 분리] 개별 차량 객체에 의존성(MQTT Client) 주입
			v := engine.Vehicle{
				ID:     id,
				Client: client,
			}

			// [안정성] 컨텍스트를 전달하여 상위(main)에서 종료 신호 시 즉시 모든 고루틴 중단 가능
			v.Start(ctx)
		}(i)
	}

	// [Graceful Shutdown] 모든 고루틴이 종료될 때까지 대기하여 데이터 유실 방지
	wg.Wait()
}
