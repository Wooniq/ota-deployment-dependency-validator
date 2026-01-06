import os
from hdbcli import dbapi
from dotenv import load_dotenv

# .env 파일에서 환경변수 로드
load_dotenv()

def populate_data():
    conn = None
    try:
        # 1. DB 연결 (환경변수 사용)
        conn = dbapi.connect(
            address=os.getenv("HANA_ADDRESS"),
            port=int(os.getenv("HANA_PORT", 30015)),
            user=os.getenv("HANA_USER"),
            password=os.getenv("HANA_PWD")
        )
        cursor = conn.cursor()

        print(">>> 기존 데이터 초기화 중...")
        # 외래키 제약 조건을 고려한 삭제 순서
        cursor.execute("DELETE FROM DEPENDENCY_RULES")
        cursor.execute("DELETE FROM UPDATE_PACKAGES")
        cursor.execute("DELETE FROM ECUS")
        cursor.execute("DELETE FROM VEHICLES")

        # 2. 테스트 차량 데이터 삽입 (리드미 status 반영)
        vehicles = [
            ("V001", "IONIQ6", "READY"),
            ("V002", "GV80", "READY")
        ]
        cursor.executemany("INSERT INTO VEHICLES (VEHICLE_ID, MODEL, STATUS) VALUES (?, ?, ?)", vehicles)

        # 3. 테스트 ECU 현황 데이터 삽입 (Semantic Versioning 구조)
        # 시나리오: BCM 1.2.0 이상이 필요한 상황
        ecus = [
            (1, "V001", "BMS", 2, 0, 0),
            (2, "V001", "BCM", 1, 5, 0),  # V001: 1.5.0 (충족)
            (3, "V002", "BMS", 2, 0, 0),
            (4, "V002", "BCM", 1, 0, 0)   # V002: 1.0.0 (미달)
        ]
        cursor.executemany("INSERT INTO ECUS (ID, VEHICLE_ID, ECU_TYPE, MAJOR_V, MINOR_V, PATCH_V) VALUES (?, ?, ?, ?, ?, ?)", ecus)

        # 4. 업데이트 패키지 데이터 삽입
        cursor.execute("INSERT INTO UPDATE_PACKAGES (PACKAGE_ID, TARGET_ECU_TYPE, TARGET_MAJOR_V, TARGET_MINOR_V, TARGET_PATCH_V) VALUES (?, ?, ?, ?, ?)",
                       ('PKG_BMS_30', 'BMS', 3, 0, 0))

        # 5. 의존성 규칙 데이터 삽입
        # 규칙: 'PKG_BMS_30' 설치 시 'BCM' 제어기가 '1.2.0' 이상이어야 함
        cursor.execute("INSERT INTO DEPENDENCY_RULES (RULE_ID, PACKAGE_ID, REQUIRED_ECU_TYPE, MIN_MAJOR_V, MIN_MINOR_V, MIN_PATCH_V) VALUES (?, ?, ?, ?, ?, ?)",
                       (101, 'PKG_BMS_30', 'BCM', 1, 2, 0))

        conn.commit()
        print("SAP HANA 샘플 데이터 주입 완료!")

    except Exception as e:
        print(f"데이터 주입 실패: {e}")
    finally:
        if conn:
            conn.close()

if __name__ == "__main__":
    populate_data()