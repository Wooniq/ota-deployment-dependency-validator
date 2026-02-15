package transport

// 임시

import (
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func NewMQTTClient(broker string) mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID("ota-agent-master")
	opts.SetAutoReconnect(true)
	opts.SetMaxReconnectInterval(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("❌ MQTT 연결 실패: %v", token.Error())
	}
	return client
}

// SendToBroker: 전송부 파라미터를 (클라이언트, 식별자, 데이터) 순으로 정리
func SendToBroker(client mqtt.Client, vin string, data []byte) {
	// 실제 MQTT 구현 시에는 client.Publish를 사용하게 됩니다.
	// 현재는 시뮬레이션을 위해 로그 출력
	log.Printf("[Transport] VIN:%s -> DLT Packet 전송 완료 (%d bytes)", vin, len(data))
}
