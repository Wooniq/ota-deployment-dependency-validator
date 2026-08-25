package service

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/Wooniq/ota-agent/pkg/repository"
)

func TestAnalyzeAndSaveBinary_SOH(t *testing.T) {
	// 1. 가상 데이터 생성 (VIN 17B + ADAS 버전 + SOH 4B)
	vin := "VIN-TEST-00000001"
	adas := "ADAS1.0.0"

	// SOH를 0.25(E1 전압 부족 상태)로 설정
	sohValue := float32(0.25)
	sohBits := math.Float32bits(sohValue)
	sohBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(sohBytes, sohBits)

	// 패킷 조립: VIN(17) + "ADAS" ID(4) + 버전 + Null(1) + SOH(4)
	payload := append([]byte(vin), []byte("ADAS")...)
	payload = append(payload, []byte(adas)...)
	payload = append(payload, 0x00)
	payload = append(payload, sohBytes...)

	// 2. 분석기 실행 (실제 DB 연결 없이 로직만 체크하기 위해 Mock이나 Dummy Repo 필요)
	// 여기서는 로직 판별 함수인 performDeepAnalysis만 직접 테스트합니다.
	analyzer := &OTAAnalyzer{
		TargetADAS: "ADAS2.0.0",
		TargetBMS:  "BMS2.0.0",
	}

	status, needsUpdate := analyzer.performDeepAnalysis("ADAS1.0.0", "BMS1.0.0", float64(sohValue))

	// 3. 검증
	if status != repository.StatusBatteryLow {
		t.Errorf("Expected E1 status for SOH 0.25, got %s", status)
	}
	if needsUpdate != false {
		t.Errorf("Expected needsUpdate to be false for E1 status")
	}
}
