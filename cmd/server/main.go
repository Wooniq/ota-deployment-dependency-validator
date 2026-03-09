package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
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

	// 2. [Repository] HANA DB 연결
	repo, err := repository.NewHANARepository(
		os.Getenv("HANA_ADDRESS"),
		os.Getenv("HANA_PORT"),
		os.Getenv("HANA_USER"),
		os.Getenv("HANA_PASSWORD"),
	)

	fmt.Println("DEBUG: HANA_ADDRESS =", os.Getenv("HANA_ADDRESS"))
	fmt.Println("DEBUG: HANA_PORT =", os.Getenv("HANA_PORT"))

	if err != nil {
		log.Fatalf("관제 시스템 구동 실패: %v", err)
	}
	defer repo.Close()
	log.Println("[Step 1] SAP HANA DB 연결 성공")

	// 3. [Transport] Kafka Producer 초기화 (3-Node 클러스터 연결)
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "kafka-1:9092,kafka-2:9092,kafka-3:9092"
	}
	brokers := strings.Split(kafkaBrokers, ",")

	// Producer 초기화 (수집용)
	kafkaProducer := transport.NewKafkaProducer(brokers, "ota-inventory")
	defer kafkaProducer.Close()
	log.Println("[Step 2] Kafka Producer (3-Node) 준비 완료")

	// 4. [Service] 서버 전용 MQTT 클라이언트 생성 및 분석 엔진 초기화
	mqttBroker := os.Getenv("MQTT_BROKER")
	if mqttBroker == "" {
		mqttBroker = "tcp://mqtt-broker:1883"
	}

	// 4-1. 서버가 명령(롤백)을 내리기 위한 MQTT 클라이언트 생성
	serverMqttClient, err := transport.NewMQTTClient(mqttBroker, "OTA-Server-Commander")
	if err != nil {
		log.Fatalf("서버 MQTT 클라이언트 초기화 실패: %v", err)
	}

	// 4-2. 파라미터 4개(repo, serverMqttClient, adasVer, bmsVer)를 정확히 전달!
	analyzer := service.NewOTAAnalyzer(repo, serverMqttClient, "v2.2.2", "v1.3.9")
	log.Println("[Step 3] OTA 분석 엔진 (Command 발송 포함) 준비 완료")

	// 5. [Transport] Kafka Consumer 기동(분석용)
	kafkaConsumer := transport.NewKafkaConsumer(brokers, "ota-inventory", "analyzer-group-v1")
	go kafkaConsumer.StartConsuming(context.Background(), analyzer)
	log.Println("[Step 4] Kafka Consumer (분석 계층) 가동 중")

	// 6. [Transport] MQTT 수집기 가동 (입력용)
	go transport.StartCollector(mqttBroker, kafkaProducer)
	log.Println("[Step 5] MQTT 수집기 가동 중 (입력 -> Kafka)")

	// 7. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("관제 시스템을 안전하게 종료합니다.")
}
