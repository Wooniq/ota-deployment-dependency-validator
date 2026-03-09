package repository

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"

	_ "github.com/SAP/go-hdb/driver"
)

// HANARepository: SAP HANA DB 연결 및 쿼리 실행 객체
type HANARepository struct {
	db *sql.DB
}

// NewHANARepository: DB 연결 및 가용성 확인
func NewHANARepository(host, port, user, password string) (*HANARepository, error) {
	dsn := fmt.Sprintf("hdb://%s:%s@%s:%s", user, password, host, port)
	db, err := sql.Open("hdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("HANA 드라이버 로드 실패: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("HANA DB 연결 실패: %w", err)
	}
	return &HANARepository{db: db}, nil
}

// BulkUpsertVehicles: 다수 차량의 상태 데이터를 일괄적으로 UPSERT 수행
func (r *HANARepository) BulkUpsertVehicles(batch []VehicleInfo) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	defer tx.Rollback()

	// 실제 DB 스키마에 맞춘 UPSERT 쿼리
	query := `UPSERT "Vehicle_ECU_Inventory" (
		"VIN",
		"ECUType",
		"HWVersion",
		"SWMajor",
		"SWMinor",
		"SWPatch",
		"BatterySOH",
		"LastReported",
		"UPDATE_STATUS",
		"REGION_CODE",
		"NEEDS_UPDATE"
	) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?, ?, ?) WITH PRIMARY KEY`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("UPSERT Prepare 실패: %w", err)
	}
	defer stmt.Close()

	for _, v := range batch {
		// 2. vehicleID 변환 로직 제거하고, v.VIN을 문자열 그대로 1번 인자로 전달
		_, err := stmt.Exec(
			v.VIN,                  // 1. "VIN" (문자열 그대로 전달)
			v.ECUType,              // 2. "ECUType"
			v.HWVersion,            // 3. "HWVersion"
			v.SWMajor,              // 4. "SWMajor"
			v.SWMinor,              // 5. "SWMinor"
			v.SWPatch,              // 6. "SWPatch"
			v.BatterySOH,           // 7. "BatterySOH"
			string(v.UpdateStatus), // 8. "UPDATE_STATUS"
			v.RegionCode,           // 9. "REGION_CODE"
			v.NeedsUpdate,          // 10. "NEEDS_UPDATE"
		)
		if err != nil {
			return fmt.Errorf("Exec 실패 (VIN: %s): %w", v.VIN, err)
		}
	}

	return tx.Commit()
}

// extractIDFromVIN: VIN 내 숫자 패턴을 BIGINT로 변환하여 PK 정합성 확보
func extractIDFromVIN(vin string) int64 {
	re := regexp.MustCompile(`[0-9]+`)
	matches := re.FindAllString(vin, -1)

	if len(matches) > 0 {
		// 명명 규칙에 따라 마지막 숫자 시퀀스를 ID로 식별
		fullDigits := matches[len(matches)-1]
		id, _ := strconv.ParseInt(fullDigits, 10, 64)
		return id
	}
	return 0
}

// Close: DB 연결 종료
func (r *HANARepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
