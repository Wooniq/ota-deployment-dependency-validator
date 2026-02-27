package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Wooniq/ota-agent/pkg/repository"
	"github.com/Wooniq/ota-agent/pkg/service"
	"github.com/Wooniq/ota-agent/pkg/transport"

	"github.com/joho/godotenv"
)

func main() {
	// 1. .env 파일 로드
	err := godotenv.Load("../../.env")
	if err != nil {
		err = godotenv.Load(".env")
	}

	// 2. [Repository] 환경 변수를 사용하여 HANA DB 연결
	repo, err := repository.NewHANARepository(
		os.Getenv("HANA_ADDRESS"),
		os.Getenv("HANA_PORT"),
		os.Getenv("HANA_USER"),
		os.Getenv("HANA_PASSWORD"),
	)
	if err != nil {
		log.Fatalf("관제 시스템 구동 실패: %v", err)
	}
	defer repo.Close()
	log.Println("[Step 1] SAP HANA DB 연결 성공")

	// 3. [Transport] Kafka Producer 초기화 (3-Node 클러스터 연결)
	brokers := []string{"localhost:29092", "localhost:29093", "localhost:29094"}
	kafkaProducer := transport.NewKafkaProducer(brokers, "ota-inventory")
	defer kafkaProducer.Close()
	log.Println("[Step 2] Kafka Producer (3-Node) 준비 완료")

	// 4. [Service] 분석 엔진 초기화
	analyzer := service.NewOTAAnalyzer(repo, "v2.2.2", "v1.3.9")
	log.Println("[Step 3] OTA 분석 엔진 준비 완료")

	// 5. [Transport] Kafka Consumer 기동
	// Kafka에서 데이터를 실시간으로 꺼내와 분석 엔진(analyzer)에 전달합니다.
	kafkaConsumer := transport.NewKafkaConsumer(brokers, "ota-inventory", "analyzer-group")
	go kafkaConsumer.StartConsuming(context.Background(), analyzer)
	log.Println("[Step 4] Kafka Consumer (분석 계층) 가동 중")

	// 6. [Transport] MQTT 수집기 가동 (Kafka Producer 주입)
	// 이제 MQTT는 분석기에 직접 주지 않고 Kafka로 보냅니다.
	broker := os.Getenv("MQTT_BROKER")
	go transport.StartCollector(broker, kafkaProducer) // analyzer 대신 kafkaProducer 전달
	log.Println("[Step 5] MQTT 수집기 가동 중 (입력 -> Kafka)")

	// 7. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("관제 시스템을 안전하게 종료합니다.")
}
