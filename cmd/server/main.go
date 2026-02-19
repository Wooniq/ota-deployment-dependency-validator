package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"ota-agent/internal/repository"
	"ota-agent/internal/service"
	"ota-agent/internal/transport"

	"github.com/joho/godotenv"
)

func main() {
	// 1. .env 파일 로드
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
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

	// 3. [Service] 분석 엔진 초기화
	// 타겟 버전 설정
	analyzer := service.NewOTAAnalyzer(repo, "v2.2.2", "v1.3.9")
	log.Println("[Step 2] OTA 분석 엔진 준비 완료")

	// 4. [Transport] MQTT 수집기 가동
	broker := os.Getenv("MQTT_BROKER")
	go transport.StartCollector(broker, analyzer)
	log.Println("[Step 3] MQTT 관제 수집 서버 가동 중 (Broker:", broker, ")")

	// 5. Graceful Shutdown
	log.Println("모든 시스템 정상 가동 중. (종료: Ctrl+C)")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("관제 시스템을 안전하게 종료합니다.")
}
