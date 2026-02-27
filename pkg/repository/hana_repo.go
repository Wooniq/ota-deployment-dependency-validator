package repository

import (
	"database/sql"
	"fmt"
	"log"

	// SAP HANA 드라이버
	_ "github.com/SAP/go-hdb/driver"
)

// HANARepository: SAP HANA DB 연결 및 쿼리 실행 객체
type HANARepository struct {
	db *sql.DB
}

// NewHANARepository: DB 연결 및 연결 상태 확인
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

// BulkUpsertVehicles: 1,000대 이상의 데이터를 고속 적재하는 핵심 함수
func (r *HANARepository) BulkUpsertVehicles(batch []VehicleInfo) error {
	if len(batch) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("트랜잭션 시작 실패: %w", err)
	}
	defer tx.Rollback()

	// [중요] SQL Error 260 해결: 실제 DB의 컬럼명과 아래 쿼리를 일치시켜야 합니다.
	// 만약 DB 컬럼명이 "VIN"이 아니라면 아래 문자열을 해당 이름으로 수정하세요.
	query := `UPSERT "Vehicle_ECU_Inventory" (
		"VIN", 
		"ECU_TYPE", 
		"SW_MAJOR_V", 
		"SW_MINOR_V", 
		"SW_PATCH_V", 
		"HW_VERSION", 
		"BATTERY_SOH", 
		"UPDATE_STATUS", 
		"LAST_REPORTED"
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP) WITH PRIMARY KEY`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("UPSERT Prepare 실패: %w", err)
	}
	defer stmt.Close()

	for _, v := range batch {
		_, err := stmt.Exec(
			v.VIN,          // 1. "VIN"
			v.ECUType,      // 2. "ECU_TYPE"
			v.SWMajor,      // 3. "SW_MAJOR_V"
			v.SWMinor,      // 4. "SW_MINOR_V"
			v.SWPatch,      // 5. "SW_PATCH_V"
			v.HWVersion,    // 6. "HW_VERSION"
			v.BatterySOH,   // 7. "BATTERY_SOH"
			v.UpdateStatus, // 8. "UPDATE_STATUS"
		)
		if err != nil {
			return fmt.Errorf("Exec 실패 (VIN: %s): %w", v.VIN, err)
		}
	}

	return tx.Commit()
}

// Close: DB 연결 안전 종료
func (r *HANARepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
