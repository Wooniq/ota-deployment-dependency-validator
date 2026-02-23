package transport

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducer : 3-Node 클러스터에 메시지를 발행하는 객체
type KafkaProducer struct {
	Writer *kafka.Writer
}

// NewKafkaProducer : 클러스터 브로커 주소들을 받아 Producer 초기화
func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	return &KafkaProducer{
		Writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{}, // 부하가 적은 브로커로 분산 전송
			MaxAttempts:  5,
			BatchSize:    100, // 1,000대 차량 대응을 위한 배치 설정
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

// PublishMessage : 분석 전의 로(Raw) 데이터를 Kafka 토픽으로 전송
func (kp *KafkaProducer) PublishMessage(ctx context.Context, key string, value []byte) error {
	err := kp.Writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("Kafka 메시지 발행 실패: %w", err)
	}
	return nil
}

// Close : 리소스 해제
func (kp *KafkaProducer) Close() error {
	return kp.Writer.Close()
}
