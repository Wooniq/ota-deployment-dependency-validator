import os
from hdbcli import dbapi
from dotenv import load_dotenv

# .env 파일에서 환경변수 로드
load_dotenv()

def populate_data():
    conn = None
    try:
        # 1. DB 연결
        conn = dbapi.connect(
            address=os.getenv("HANA_ADDRESS"),
            port=int(os.getenv("HANA_PORT", 39013)), # 포트 번호 주의 (Express는 보통 39013)
            user=os.getenv("HANA_USER"),
            password=os.getenv("HANA_PASSWORD")
        )
        cursor = conn.cursor()

        print(">>> 기존 데이터 초기화 중...")
        # 삭제 순서 (외래키 역순)
        cursor.execute('DELETE FROM "Update_History"')
        cursor.execute('DELETE FROM "Dependency_Rules"')
        cursor.execute('DELETE FROM "Deployment_Campaigns"')
        cursor.execute('DELETE FROM "Update_Packages"')
        cursor.execute('DELETE FROM "Vehicle_ECU_Inventory"')
        cursor.execute('DELETE FROM "Vehicles"')

        # 2. 테스트 차량 데이터 (VIN 기반)
        vehicles = [
            (1, "KMHGN7HG1PA123456", "GN7", "KR"),
            (2, "KMHNQ5LG2NA654321", "NQ5", "KR")
        ]
        cursor.executemany('INSERT INTO "Vehicles" (VEHICLE_ID, VIN, MODEL_CODE, REGION) VALUES (?, ?, ?, ?)', vehicles)

        # 3. 제어기 인벤토리 (현재 상태)
        # 시나리오: BCM 버전이 1.2.0 이상이어야 함
        inventory = [
            (1, "BMS", "HW_1.0", 2, 0, 0, '2024-12-01 10:00:00'),
            (1, "BCM", "HW_1.1", 1, 5, 0, '2024-12-01 10:00:00'), # GN7: 1.5.0 (충족)
            (2, "BMS", "HW_1.0", 2, 0, 0, '2024-12-01 11:00:00'),
            (2, "BCM", "HW_1.1", 1, 0, 0, '2024-12-01 11:00:00')  # NQ5: 1.0.0 (미달)
        ]
        cursor.executemany('INSERT INTO "Vehicle_ECU_Inventory" (VEHICLE_ID, ECU_TYPE, HW_VERSION, SW_MAJOR_V, SW_MINOR_V, SW_PATCH_V, LAST_REPORTED_AT) VALUES (?, ?, ?, ?, ?, ?, ?)', inventory)

        # 4. 배포 패키지 (목표 SW)
        # 패키지: BMS 3.0.0 (HW_1.0 용)
        cursor.execute('INSERT INTO "Update_Packages" (PACKAGE_ID, TARGET_ECU_TYPE, TARGET_HW_VERSION, SW_MAJOR_V, SW_MINOR_V, SW_PATCH_V, FILE_HASH) VALUES (?, ?, ?, ?, ?, ?, ?)',
                       ('PKG_BMS_30', 'BMS', 'HW_1.0', 3, 0, 0, 'SHA256_HASH_SAMPLE_123'))

        # 5. 캠페인 생성 (중요: 이게 있어야 View에서 조회가 됨!)
        cursor.execute('INSERT INTO "Deployment_Campaigns" (CAMPAIGN_ID, PACKAGE_ID, TARGET_MODEL_CODE, STATUS, START_DATE, END_DATE) VALUES (?, ?, ?, ?, ?, ?)',
                       (1001, 'PKG_BMS_30', 'GN7', 'ACTIVE', '2024-01-01 00:00:00', '2026-12-31 23:59:59'))
        cursor.execute('INSERT INTO "Deployment_Campaigns" (CAMPAIGN_ID, PACKAGE_ID, TARGET_MODEL_CODE, STATUS, START_DATE, END_DATE) VALUES (?, ?, ?, ?, ?, ?)',
                       (1002, 'PKG_BMS_30', 'NQ5', 'ACTIVE', '2024-01-01 00:00:00', '2026-12-31 23:59:59'))

        # 6. 의존성 규칙
        # PKG_BMS_30을 깔려면 BCM이 1.2.0 이상이어야 함
        cursor.execute('INSERT INTO "Dependency_Rules" (RULE_ID, PACKAGE_ID, REQUIRED_ECU_TYPE, MIN_SW_MAJOR_V, MIN_SW_MINOR_V, MIN_SW_PATCH_V) VALUES (?, ?, ?, ?, ?, ?)',
                       (5001, 'PKG_BMS_30', 'BCM', 1, 2, 0))

        conn.commit()
        print("[success] Pro-level 스키마 기반 샘플 데이터 주입 완료!")

    except Exception as e:
        print(f"[failed] 데이터 주입 중 오류 발생: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    populate_data()