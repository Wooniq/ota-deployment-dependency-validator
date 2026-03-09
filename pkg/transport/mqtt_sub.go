package transport

// 서버(Cloud) 입장에서 브로커에 쌓이는 1,000대의 데이터를 받는(Subscribe) 역할
// Autosar DLT 표준 바이너리 규격 수집기

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/Wooniq/ota-agent/pkg/protocol"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// StartCollector : 모든 차량의 인벤토리 토픽을 구독하여 Kafka로 중계
func StartCollector(brokerAddr string, kp *KafkaProducer) { // analyzer 대신 kp 전달
	opts := mqtt.NewClientOptions().AddBroker(brokerAddr)
	opts.SetClientID("ota-server-bridge")
	opts.SetAutoReconnect(true)

	opts.OnConnect = func(c mqtt.Client) {
		log.Println("[MQTT-Bridge] 가동. 수신 데이터를 Kafka 3-Node 클러스터로 전달합니다.")

		// 1,000대 차량 토픽 구독
		token := c.Subscribe("ota/vehicles/+/inventory", 1, func(client mqtt.Client, msg mqtt.Message) {
			// 1. 토픽에서 VIN 추출
			topicParts := strings.Split(msg.Topic(), "/")
			if len(topicParts) < 3 {
				return
			}
			vin := topicParts[2]

			// 2. [표준 규격] DLT 패킷 디코딩
			rawPayload := msg.Payload()
			_, err := protocol.ParseDltPacket(rawPayload)
			if err != nil {
				log.Printf("[Protocol-Error] VIN:%s 비표준 패킷 무시: %v", vin, err)
				return
			}
			log.Printf("[MQTT-Bridge] 데이터 수신 성공! VIN: %s, Data: %s", vin, string(rawPayload))

			// 3. [ADR 0001] Kafka Producer를 통해 클러스터 적재
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			//  dltData가 아닌 원본 데이터(rawPayload)를 Kafka에 그대로 적재
			if err := kp.PublishMessage(ctx, vin, rawPayload); err != nil {
				log.Printf("[Kafka-Error] VIN:%s 전송 실패: %v", vin, err)
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
