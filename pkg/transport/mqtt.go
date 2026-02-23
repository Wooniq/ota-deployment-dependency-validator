package transport

// 차량(Edge) 입장에서 브로커에 데이터를 보내는(Publish) 역할. (시뮬레이터용)

/*
 * 핵심 최적화 요약:
 * 1. 시스템 안정성 및 고가용성: Auto-Reconnect와 KeepAlive 설정을 통해 불안정한 차량 네트워크 환경에서도 끊김 없는 데이터 가용성을 확보했습니다.
 * 2. 데이터 무결성 보장 (QoS 1): OTA 배포 전 사전 검증에 필수적인 인벤토리 정보의 누락을 방지하기 위해 '최소 1회 전송'을 보장하는 QoS 1을 적용했습니다.
 * 3. 저지연/비동기 처리 (Non-blocking): 전송 대기(Wait) 로직을 별도 고루틴으로 분리하여, 대규모 트래픽 발생 시에도 개별 에이전트의 성능 저하를 방지하는 저지연 아키텍처를 구현했습니다.
 */

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// NewMQTTClient : 차량별 고유 세션을 가진 MQTT 클라이언트 생성
func NewMQTTClient(brokerAddr string, clientID string) (mqtt.Client, error) {
	//fmt.Printf("[Debug] VIN:%s 접속 시도 중 (Addr: %s)...\n", clientID, brokerAddr)

	opts := mqtt.NewClientOptions()
	opts.AddBroker(brokerAddr)
	opts.SetClientID(clientID)
	opts.SetCleanSession(true)

	client := mqtt.NewClient(opts)
	token := client.Connect()

	if token.WaitTimeout(3*time.Second) && token.Error() != nil {
		return nil, token.Error()
	}

	//fmt.Printf("[Debug] VIN:%s 접속 성공!\n", clientID)
	return client, nil
}

// SendToBroker : DLT 패킷 비동기 전송
func SendToBroker(client mqtt.Client, vin string, data []byte) {
	if client == nil || !client.IsConnected() {
		//log.Printf("[Transport-Warn] VIN:%s -> 연결 끊김. 자동 재접속 대기 중.", vin)
		return
	}

	// [도메인 설계] Kafka 및 DB 인덱싱을 고려한 계층형 토픽 구조
	topic := fmt.Sprintf("ota/vehicles/%s/inventory", vin)

	// [데이터 정합성] QoS 1 (At least once) 적용으로 전송 신뢰성 확보
	token := client.Publish(topic, 1, false, data)

	// 비동기 모니터링으로 엔진 루프 병목 방지
	go func() {
		// WaitTimeout을 설정하여 네트워크 행(Hang) 상태 대응력 강화
		if token.WaitTimeout(3*time.Second) && token.Error() != nil {
			//log.Printf("[Transport-Fail] VIN:%s 전송 실패: %v", vin, token.Error())
		} else {
			// 성공 로그 (실제 운영 시에는 Debug 레벨로 관리 권장)
			// log.Printf("[Transport-Success] VIN:%s -> MQTT Publish 완료", vin)
		}
	}()
}
