package transport

// 서버(Cloud) 입장에서 브로커에 쌓이는 1,000대의 데이터를 받는(Subscribe) 역할
// Autosar DLT 표준 바이너리 규격 수집기

import (
	"log"
	"strings"

	"ota-agent/internal/protocol"
	"ota-agent/internal/service"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// StartCollector : 모든 차량의 인벤토리 토픽을 구독하는 수집기 가동
func StartCollector(brokerAddr string, analyzer *service.OTAAnalyzer) {
	opts := mqtt.NewClientOptions().AddBroker(brokerAddr)
	opts.SetClientID("ota-server-subscriber")
	opts.SetAutoReconnect(true)

	// 연결 성공 시 자동 구독 설정
	opts.OnConnect = func(c mqtt.Client) {
		log.Println("[MQTT-Sub] 표준 DLT 수집기 가동. 인벤토리 수집 시작...")

		// 와일드카드(+)를 사용하여 1,000대 차량 토픽 구독
		token := c.Subscribe("ota/vehicles/+/inventory", 1, func(client mqtt.Client, msg mqtt.Message) {
			// 1. 토픽에서 VIN 추출
			topicParts := strings.Split(msg.Topic(), "/")
			if len(topicParts) < 3 {
				return
			}
			vin := topicParts[2]

			// 2. [표준 규격] 바이너리 페이로드 직접 수집
			// string() 변환은 바이너리 데이터를 깨뜨릴 수 있으므로 원본 slice를 사용함
			rawPayload := msg.Payload()

			// 3. [표준 규격] DLT 패킷 디코딩
			// protocol 패키지의 ParseDltPacket을 호출하여 헤더를 검증하고 순수 페이로드를 분리
			dltData, err := protocol.ParseDltPacket(rawPayload)
			if err != nil {
				log.Printf("[Protocol-Error] VIN:%s 비표준 패킷 수신: %v", vin, err)
				return
			}

			// 4. 분석 및 SAP HANA DB 저장 서비스 호출
			// 기존 AnalyzeAndSave 대신 바이너리 데이터를 처리하는 메서드 호출
			if err := analyzer.AnalyzeAndSaveBinary(vin, dltData); err != nil {
				log.Printf("[Analysis-Error] VIN:%s 처리 실패: %v", vin, err)
			}
		})

		if token.Wait() && token.Error() != nil {
			log.Fatalf("구독 신청 실패: %v", token.Error())
		}
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT 브로커 연결 실패: %v", token.Error())
	}
}
