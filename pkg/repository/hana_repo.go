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

	// 스키마(Level 1) 규격에 맞춘 UPSERT 쿼리
	query := `UPSERT "Vehicle_ECU_Inventory" (
		"VEHICLE_ID", 
		"ECU_TYPE", 
		"HW_VERSION", 
		"SW_MAJOR_V", 
		"SW_MINOR_V", 
		"SW_PATCH_V", 
		"BATTERY_SOH", 
		"LAST_REPORTED_AT", 
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
		// VIN 문자열에서 마지막 숫자 그룹을 추출하여 BIGINT 타입의 VEHICLE_ID 생성
		vehicleID := extractIDFromVIN(v.VIN)

		// 2. Exec 인자 전달
		_, err := stmt.Exec(
			vehicleID,    // 1. VEHICLE_ID (BIGINT)
			v.ECUType,    // 2. ECU_TYPE
			v.HWVersion,  // 3. HW_VERSION
			v.SWMajor,    // 4. SW_MAJOR_V
			v.SWMinor,    // 5. SW_MINOR_V
			v.SWPatch,    // 6. SW_PATCH_V
			v.BatterySOH, // 7. BATTERY_SOH
			// CURRENT_TIMESTAMP 자리 (인자 전달 필요 없음)
			string(v.UpdateStatus), // 8. UPDATE_STATUS
			v.RegionCode,           // 9. REGION_CODE
			v.NeedsUpdate,          // 10. NEEDS_UPDATE
		)
		if err != nil {
			return fmt.Errorf("Exec 실패 (ID: %d): %w", vehicleID, err)
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
