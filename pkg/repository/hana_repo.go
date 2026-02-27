package repository

// HANA DB와의 연결과 실제 쿼리 담당

import (
	"database/sql"
	"fmt"

	// SAP HANA 드라이버 (런타임 로드용)
	_ "github.com/SAP/go-hdb/driver"
)

// HANARepository : SAP HANA 데이터베이스와의 통신을 담당하는 객체
type HANARepository struct {
	db *sql.DB
}

// NewHANARepository : HANA DB 연결 객체를 생성하고 가용성 검증(Ping)
func NewHANARepository(host, port, user, password string) (*HANARepository, error) {
	// DSN 형식: hdb://user:password@host:port
	dsn := fmt.Sprintf("hdb://%s:%s@%s:%s", user, password, host, port)

	db, err := sql.Open("hdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("HANA 드라이버 로드 실패: %w", err)
	}

	// [Reliability Check] 시스템 가동 전 실제 DB 연결이 유효한지 즉시 검증
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("HANA DB 응답 없음(Ping 실패): %w", err)
	}

	return &HANARepository{db: db}, nil
}

// GetAllVehicles : DB에 저장된 모든 차량의 인벤토리 정보를 조회
func (r *HANARepository) GetAllVehicles() ([]VehicleInfo, error) {
	query := `SELECT VIN, HW_VERSION, ADAS_VERSION, BMS_VERSION, UPDATE_STATUS, REGION_CODE, BATTERY_SOH, NEEDS_UPDATE FROM VEHICLE_ECU_INVENTORY`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("차량 목록 조회 실패: %w", err)
	}
	defer rows.Close()

	var list []VehicleInfo
	for rows.Next() {
		var v VehicleInfo
		if err := rows.Scan(&v.VIN, &v.HWVersion, &v.ADASVersion, &v.BMSVersion, &v.UpdateStatus, &v.RegionCode, &v.BatterySOH, &v.NeedsUpdate); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

// GetVehiclesByUpdateStatus : 업데이트 필요 여부(NeedsUpdate)에 따라 차량 필터링 조회
func (r *HANARepository) GetVehiclesByUpdateStatus(needsUpdate bool) ([]VehicleInfo, error) {
	query := `SELECT VIN, HW_VERSION, ADAS_VERSION, BMS_VERSION, UPDATE_STATUS, REGION_CODE, BATTERY_SOH, NEEDS_UPDATE 
              FROM VEHICLE_ECU_INVENTORY WHERE NEEDS_UPDATE = ?`

	rows, err := r.db.Query(query, needsUpdate)
	if err != nil {
		return nil, fmt.Errorf("필터링 조회 실패: %w", err)
	}
	defer rows.Close()

	var list []VehicleInfo
	for rows.Next() {
		var v VehicleInfo
		if err := rows.Scan(&v.VIN, &v.HWVersion, &v.ADASVersion, &v.BMSVersion, &v.UpdateStatus, &v.RegionCode, &v.BatterySOH, &v.NeedsUpdate); err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	return list, nil
}

// UpsertVehicle : 차량 인벤토리 정보를 저장하거나 기존 정보 업데이트(Merge)
func (r *HANARepository) UpsertVehicle(info VehicleInfo) error {
	query := `
       MERGE INTO VEHICLE_ECU_INVENTORY AS target
	   USING (
		  SELECT 
			 ? AS VIN, 
			 ? AS HW_VER, 
			 ? AS ADAS_VER, 
			 ? AS BMS_VER, 
			 ? AS UPD_STATUS,
			 ? AS REG_CODE,
			 CAST(? AS DOUBLE) AS BATT_SOH,  -- 명시적 타입 변환 추가
			 CAST(? AS BOOLEAN) AS NEEDS_UPD -- 명시적 타입 변환 추가
		  FROM DUMMY
	   ) AS source
       ON target.VIN = source.VIN
       WHEN MATCHED THEN
          UPDATE SET 
             HW_VERSION = source.HW_VER,       -- DB 컬럼명 확인
             ADAS_VERSION = source.ADAS_VER,   -- DB 컬럼명 확인
             BMS_VERSION = source.BMS_VER,     -- DB 컬럼명 확인
             UPDATE_STATUS = source.UPD_STATUS,
             REGION_CODE = source.REG_CODE,    -- DB 컬럼명 확인
             BATTERY_SOH = source.BATT_SOH,    -- 에러 발생 지점 교정
             NEEDS_UPDATE = source.NEEDS_UPD,
             LAST_REPORTED = CURRENT_TIMESTAMP
       WHEN NOT MATCHED THEN
          INSERT (
             VIN, HW_VERSION, ADAS_VERSION, BMS_VERSION, 
             UPDATE_STATUS, REGION_CODE, BATTERY_SOH, 
             NEEDS_UPDATE, LAST_REPORTED
          )
          VALUES (
             source.VIN, source.HW_VER, source.ADAS_VER, source.BMS_VER, 
             source.UPD_STATUS, source.REG_CODE, source.BATT_SOH, 
             source.NEEDS_UPD, CURRENT_TIMESTAMP
          )`

	// 파라미터 매핑 순서 확인
	_, err := r.db.Exec(query,
		info.VIN,
		info.HWVersion,
		info.ADASVersion,
		info.BMSVersion,
		info.UpdateStatus,
		info.RegionCode,
		info.BatterySOH,
		info.NeedsUpdate,
	)

	if err != nil {
		return fmt.Errorf("HANA Upsert 실패 (VIN: %s): %w", info.VIN, err)
	}
	return nil
}

// Close : 서버 종료 시 DB 커넥션 풀을 안전하게 닫음
func (r *HANARepository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// BulkUpsertVehicles : 대량의 차량 상태 데이터를 HANA DB에 고속 적재합니다.
func (r *HANARepository) BulkUpsertVehicles(batch []VehicleInfo) error {
	if len(batch) == 0 {
		return nil
	}

	// 1. 트랜잭션 시작
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("HANA 트랜잭션 시작 실패: %w", err)
	}
	// 에러 발생 시 롤백 (정상 Commit 시에는 영향 없음)
	defer tx.Rollback()

	// 2. SAP HANA 전용 UPSERT 구문 준비 (Vehicle_ECU_Inventory 테이블 대상)
	query := `UPSERT "Vehicle_ECU_Inventory" (
       "VEHICLE_ID", 
       "ECU_TYPE", 
       "HW_VERSION", 
       "SW_MAJOR_V", 
       "BATTERY_SOH", 
       "LAST_REPORTED_AT"
    ) VALUES (?, ?, ?, ?, ?, ?) WITH PRIMARY KEY`

	// 3. 구문 준비 (tx.Prepare)
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("UPSERT 구문 Prepare 실패: %w", err)
	}
	defer stmt.Close()

	// 3. 루프를 돌며 Batch 실행
	for _, v := range batch {
		// VIN 대신 VEHICLE_ID를 사용해야 하므로, 실무에서는 VIN으로 ID를 조회하는 로직이 필요하지만
		// 시뮬레이션 효율을 위해 VIN의 뒷자리 숫자를 ID로 변환하여 예시를 작성합니다.
		vehicleID := extractIDFromVIN(v.VIN)

		_, err := stmt.Exec(
			vehicleID,
			"ADAS", // 혹은 제어기 타입에 맞는 로직
			v.HWVersion,
			parseMajorVersion(v.ADASVersion),
			v.BatterySOH,
			v.LastReported,
		)
		if err != nil {
			return fmt.Errorf("Exec 실패 (VIN: %s): %w", v.VIN, err)
		}
	}

	// 4. 최종 커밋
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("HANA 트랜잭션 커밋 실패: %w", err)
	}

	return nil
}

// 간단한 도우미 함수들 (실제 로직에 맞게 수정 필요)
func extractIDFromVIN(vin string) int64 {
	// VIN 문자열에서 숫자만 추출하거나 해시화하여 BIGINT ID 생성
	// 여기서는 시뮬레이션 편의를 위해 뒷자리 사용 예시
	var id int64
	fmt.Sscanf(vin[len(vin)-4:], "%d", &id)
	return id
}

func parseMajorVersion(version string) int {
	// "v2.2.2" -> 2 추출 로직
	var major int
	fmt.Sscanf(version, "v%d", &major)
	return major
}
