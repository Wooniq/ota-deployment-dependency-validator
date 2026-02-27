package repository

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv" // VIN 변환용

	_ "github.com/SAP/go-hdb/driver"
)

type HANARepository struct {
	db *sql.DB
}

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

func (r *HANARepository) BulkUpsertVehicles(batch []VehicleInfo) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	defer tx.Rollback()

	query := `UPSERT "Vehicle_ECU_Inventory" (
		"VEHICLE_ID", 
		"ECU_TYPE", 
		"HW_VERSION", 
		"SW_MAJOR_V", 
		"SW_MINOR_V", 
		"SW_PATCH_V", 
		"BATTERY_SOH", 
		"BATTERY_VOLTAGE", 
		"LAST_REPORTED_AT"
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP) WITH PRIMARY KEY`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("UPSERT Prepare 실패: %w", err)
	}
	defer stmt.Close()

	for _, v := range batch {
		// VIN(문자열)에서 마지막 숫자들을 추출해 VEHICLE_ID(BIGINT) 생성
		vehicleID := extractIDFromVIN(v.VIN)

		_, err := stmt.Exec(
			vehicleID,    // 1. VEHICLE_ID (BIGINT)
			v.ECUType,    // 2. ECU_TYPE
			v.HWVersion,  // 3. HW_VERSION
			v.SWMajor,    // 4. SW_MAJOR_V
			v.SWMinor,    // 5. SW_MINOR_V
			v.SWPatch,    // 6. SW_PATCH_V
			v.BatterySOH, // 7. BATTERY_SOH
			12.6,         // 8. BATTERY_VOLTAGE (임시 기본값)
		)
		if err != nil {
			return fmt.Errorf("Exec 실패 (ID: %d): %w", vehicleID, err)
		}
	}

	return tx.Commit()
}

// extractIDFromVIN: 문자열에서 숫자만 추출하여 BIGINT로 변환
func extractIDFromVIN(vin string) int64 {
	// 정규식을 사용하여 문자열 내의 숫자 부분만 추출합니다.
	re := regexp.MustCompile(`[0-9]+`)
	matches := re.FindAllString(vin, -1)

	if len(matches) > 0 {
		// 마지막 숫자 그룹(예: 0001)을 가져와 변환
		fullDigits := matches[len(matches)-1]
		id, _ := strconv.ParseInt(fullDigits, 10, 64)
		return id
	}
	return 0
}

func (r *HANARepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
