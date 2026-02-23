package main

import (
	"log"
	"net/http"
	"os"

	"github.com/Wooniq/ota-agent/pkg/repository"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 1. 환경 변수에서 DB 정보 읽기
	err := godotenv.Load(".env")
	if err != nil {
		log.Println("[ERROR] .env 파일을 찾을 수 없어 기본 환경 변수 사용")
	}

	addr := os.Getenv("HANA_ADDRESS")
	port := os.Getenv("HANA_PORT")
	user := os.Getenv("HANA_USER")
	pass := os.Getenv("HANA_PASSWORD")

	// 2. DB 연결
	repo, err := repository.NewHANARepository(addr, port, user, pass)
	if err != nil {
		log.Fatalf("Dashboard 연결 실패: %v", err)
	}
	defer repo.Close()

	r := gin.Default()

	// 3. API 라우팅 그룹 설정
	api := r.Group("/api/v1")
	{
		// 전체 차량 리스트 조회
		api.GET("/vehicles", func(c *gin.Context) {
			vehicles, err := repo.GetAllVehicles()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  "error",
					"message": "데이터 조회 중 오류 발생",
					"details": err.Error(),
				})
				return
			}
			c.JSON(http.StatusOK, vehicles)
		})

		// 업데이트가 필요한(NeedsUpdate=true) 차량만 필터링
		api.GET("/updates", func(c *gin.Context) {
			vehicles, err := repo.GetVehiclesByUpdateStatus(true)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"status":  "error",
					"message": "업데이트 대상 필터링 실패",
				})
				return
			}
			c.JSON(http.StatusOK, vehicles)
		})
	}

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("서버 시작 실패: %v", err)
	}
}
