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
