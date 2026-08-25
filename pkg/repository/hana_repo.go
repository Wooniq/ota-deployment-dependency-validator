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

// GetAllVehicles: Vehicle_ECU_Inventory 테이블의 전체 차량/ECU 목록 조회
func (r *HANARepository) GetAllVehicles() ([]VehicleInfo, error) {
	query := `SELECT
		"VIN", "ECUType", "HWVersion", "SWMajor", "SWMinor", "SWPatch",
		"BatterySOH", "LastReported", "UPDATE_STATUS", "REGION_CODE", "NEEDS_UPDATE"
	FROM "Vehicle_ECU_Inventory"`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("전체 차량 조회 실패: %w", err)
	}
	defer rows.Close()

	return scanVehicleRows(rows)
}

// GetVehiclesByUpdateStatus: NEEDS_UPDATE 값으로 필터링된 차량/ECU 목록 조회
func (r *HANARepository) GetVehiclesByUpdateStatus(needsUpdate bool) ([]VehicleInfo, error) {
	query := `SELECT
		"VIN", "ECUType", "HWVersion", "SWMajor", "SWMinor", "SWPatch",
		"BatterySOH", "LastReported", "UPDATE_STATUS", "REGION_CODE", "NEEDS_UPDATE"
	FROM "Vehicle_ECU_Inventory"
	WHERE "NEEDS_UPDATE" = ?`

	rows, err := r.db.Query(query, needsUpdate)
	if err != nil {
		return nil, fmt.Errorf("업데이트 대상 차량 조회 실패: %w", err)
	}
	defer rows.Close()

	return scanVehicleRows(rows)
}

// scanVehicleRows: Vehicle_ECU_Inventory 쿼리 결과를 VehicleInfo 슬라이스로 스캔
func scanVehicleRows(rows *sql.Rows) ([]VehicleInfo, error) {
	vehicles := make([]VehicleInfo, 0)
	for rows.Next() {
		var v VehicleInfo
		var status string
		if err := rows.Scan(
			&v.VIN, &v.ECUType, &v.HWVersion, &v.SWMajor, &v.SWMinor, &v.SWPatch,
			&v.BatterySOH, &v.LastReported, &status, &v.RegionCode, &v.NeedsUpdate,
		); err != nil {
			return nil, fmt.Errorf("차량 정보 스캔 실패: %w", err)
		}
		v.UpdateStatus = StatusCode(status)
		vehicles = append(vehicles, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("차량 조회 결과 순회 실패: %w", err)
	}
	return vehicles, nil
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

// 특정 차량/ECU의 가장 최근 성공(StatusSuccess) 버전 조회 (현재는 Mock 연동)
func (r *HANARepository) GetLastStableFirmware(vin string, ecuType string) (version string, path string, hash string, err error) {
	/* 나중에 실제 DB 연동할 때 주석 해제
	query := `
		SELECT "SWVersion", "FilePath", "ExpectedHash"
		FROM "Update_History"
		WHERE "VIN" = ? AND "ECUType" = ? AND "Status" = 'StatusSuccess'
		ORDER BY "Timestamp" DESC
		LIMIT 1
	`
	row := r.db.QueryRow(query, vin, ecuType)
	err = row.Scan(&version, &path, &hash)
	*/

	// db.QueryRow 등을 이용한 실제 DB 연동 로직
	// row := r.db.QueryRow(query, vin, ecuType)
	// err = row.Scan(&version, &path, &hash)

	// TODO: DB 연동 전 임시 모의(Mock) 데이터 반환
	return "v2.1.0", "/firmware/" + ecuType + "/v2.1.0.bin", "a1b2c3d4e5f6dummyhash", nil
}

// 차량의 롤백 및 업데이트 이력 저장 (대시보드 트래킹 용도)
func (r *HANARepository) RecordUpdateHistory(vin string, ecuType string, status string, targetVersion string) error {
	/* 나중에 실제 DB 연동할 때 주석 해제
	query := `
		INSERT INTO "Update_History" ("VIN", "ECUType", "Status", "TargetVersion", "Timestamp")
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	_, err := r.db.Exec(query, vin, ecuType, status, targetVersion)
	return err
	*/

	// TODO: DB 연동 전 임시 성공 반환
	return nil
}

// Close: DB 연결 종료
func (r *HANARepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
