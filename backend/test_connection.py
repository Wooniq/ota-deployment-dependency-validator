import os
from hdbcli import dbapi

# 1. HANA DB 연결 설정 (추후 .env 파일로 관리)
HANA_HOST = 'your-hana-host'
HANA_PORT = 30015 # 기본 포트
HANA_USER = 'DBADMIN'
HANA_PWD = 'your-password'

def populate_data():
    try:
        conn = dbapi.connect(address=HANA_HOST, port=HANA_PORT, user=HANA_USER, password=HANA_PWD)
        cursor = conn.cursor()

        # 데이터 초기화 (기존 데이터 삭제)
        cursor.execute("DELETE FROM ota_tables.Vehicles")
        
        # 2. 테스트 차량 데이터 삽입
        vehicles = [
            ("V001", "IONIQ6", "READY"),   # 업데이트 성공 예정 차량
            ("V002", "GV80", "READY")      # 버전 미달로 실패 예정 차량
        ]
        cursor.executemany("INSERT INTO ota_tables.Vehicles VALUES (?, ?, ?)", vehicles)

        # 3. 테스트 ECU 현황 데이터 삽입 (V002는 BCM 버전이 낮음)
        ecus = [
            (1, "V001", "BMS", 2, 0, 0),
            (2, "V001", "BCM", 1, 5, 0), # V001의 BCM은 1.5.0
            (3, "V002", "BMS", 2, 0, 0),
            (4, "V002", "BCM", 1, 0, 0)  # V002의 BCM은 1.0.0 (실패 유도용)
        ]
        cursor.executemany("INSERT INTO ota_tables.ECUs VALUES (?, ?, ?, ?, ?, ?)", ecus)

        # 4. 업데이트 패키지 및 의존성 규칙
        cursor.execute("INSERT INTO ota_tables.UpdatePackages VALUES ('PKG_BMS_30', 'BMS', 3, 0, 0)")
        
        # 규칙: BMS 3.0을 깔려면 BCM이 최소 1.2.0 이상이어야 함
        cursor.execute("INSERT INTO ota_tables.DependencyRules VALUES (101, 'PKG_BMS_30', 'BCM', 1, 2, 0)")

        conn.commit()
        print("샘플 데이터 삽입 완료")

    except Exception as e:
        print(f"에러 발생: {e}")
    finally:
        conn.close()

if __name__ == "__main__":
    populate_data()