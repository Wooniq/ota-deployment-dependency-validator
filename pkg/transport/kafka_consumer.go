package transport

import (
	"context"
	"log"
	"time"

	"github.com/Wooniq/ota-agent/pkg/service"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	Reader *kafka.Reader
}

// NewKafkaConsumer : 컨슈머 초기화 (그룹 ID를 지정하여 메시지 분산 처리)
func NewKafkaConsumer(brokers []string, topic, groupID string) *KafkaConsumer {
	return &KafkaConsumer{
		Reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			GroupID:        groupID, // 동일 그룹 내 컨슈머끼리 메시지 분배
			Topic:          topic,
			MinBytes:       1,
			MaxBytes:       10e6, // 10MB
			StartOffset:    kafka.FirstOffset,
			CommitInterval: time.Second,
			// 연결 문제 발생 시 더 빨리 알 수 있도록 재시도 간격 조정
          		MaxWait:        500 * time.Millisecond,
		}),
	}
}

// 비동기 (고루틴 활용) 분석
func (kc *KafkaConsumer) StartConsuming(ctx context.Context, analyzer *service.OTAAnalyzer) {
	defer kc.Reader.Close()
	log.Println("[Kafka] Consumer 가동 시작...")

	for {
		m, err := kc.Reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("[Error] 메시지 읽기 실패: %v", err)
			break
		}

		// 핵심 수정: 각 메시지 처리를 고루틴으로 분리 (비동기 처리)
		go func(msg kafka.Message) {
			vin := string(msg.Key)
			// 만약 Key가 비어있다면 Payload에서 VIN을 추출하는 로직이 필요할 수 있습니다.
			if err := analyzer.AnalyzeAndSaveBinary(vin, msg.Value); err != nil {
				log.Printf("[Error] 데이터 분석/저장 실패 (VIN: %s): %v", vin, err)
			}
		}(m)
	}
}
