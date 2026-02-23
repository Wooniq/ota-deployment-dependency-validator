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
			MinBytes:       10e3, // 10KB
			MaxBytes:       10e6, // 10MB
			CommitInterval: time.Second,
		}),
	}
}

// StartConsuming : 무한 루프를 돌며 메시지를 가져와 분석기로 전달
func (kc *KafkaConsumer) StartConsuming(ctx context.Context, analyzer *service.OTAAnalyzer) {
	defer kc.Reader.Close()

	log.Println("[Kafka] Consumer 가동 시작...")

	for {
		m, err := kc.Reader.ReadMessage(ctx)
		if err != nil {
			log.Printf("[Error] 메시지 읽기 실패: %v", err)
			break
		}

		log.Printf("[Kafka] 메시지 수신 - Topic: %s, Partition: %d, Offset: %d", m.Topic, m.Partition, m.Offset)

		// [ADR 0001] 분석 엔진 호출 (바이너리 데이터 해석 및 E1 체크)
		vin := string(m.Key)
		if err := analyzer.AnalyzeAndSaveBinary(vin, m.Value); err != nil {
			log.Printf("[Error] 데이터 분석/저장 실패 (VIN: %s): %v", vin, err)
		}
	}
}
