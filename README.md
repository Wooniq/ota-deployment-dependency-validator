# OTA Deployment Dependency Validation System

> **SAP HANA 기반 OTA 배포 사전 검증 시스템**  
> ECU 간 소프트웨어 버전 의존성을 DB 레벨에서 검증하여  
> OTA 업데이트 실패(Brick)를 사전에 방지하는 관리 플랫폼

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Project Background

차량 OTA(Over-The-Air) 업데이트에서 가장 치명적인 문제는 **제어기(ECU) 간 버전 의존성 미검증으로 인한 업데이트 실패(Brick)** 입니다.

본 프로젝트는 현대자동차 SDV 환경을 가정하여, **리눅스 에이전트 기반의 실시간 데이터 수집**부터 **SAP HANA DB 레벨의 사전 검증**, 그리고 **운영 관제 대시보드**까지의 End-to-End 흐름을 구현합니다.

> OTA 배포 이전에 "이 차량은 업데이트 가능한 상태인가?"를 **DB 레벨에서 명확히 판단**하는 것이 핵심입니다.

## System Architecture


1. **Vehicle Edge (Linux)**: `ECU Agent`가 `/etc/ecu_info`를 파싱하여 현재 버전을 API로 전송
2. **OTA Server (FastAPI)**: 차량 데이터를 수집하고 SAP HANA DB에 상태 업데이트
3. **Database (SAP HANA)**: `Dependency Rules`를 기반으로 CDS View를 통한 실시간 의존성 검증
4. **Monitoring (Dashboard)**: 전체 Fleet의 업데이트 가용성 및 Brick 위험도 시각화

### Key Objectives
- ✅ ECU 간 **SW 버전 의존성 모델링**
- ✅ OTA 패키지 배포 가능 여부를 **DB에서 사전 판단**
- ✅ Brick 방지를 위한 **Pre-condition Check** 구조 설계
- ✅ SAP HANA의 **In-Memory 고속 조인** 활용

> OTA 배포 이전에 "이 차량은 업데이트 가능한 상태인가?"를  
> **Yes / No로 명확히 판단**하는 것이 핵심입니다.

---

## Why SAP HANA?

### 현대자동차 기술 스택 경험
본 프로젝트는 현대자동차그룹의 실제 엔터프라이즈 환경을 이해하기 위해 SAP HANA DB를 선택했습니다.
차량 인벤토리 데이터는 쓰기(Write)보다 조회(Read)와 복잡한 조인(Join) 연산이 많기 때문에, HANA의 Column Storage 방식이 OTA 적합성 검증에 최적이라고 판단했습니다.

### HANA의 기술적 장점 (OTA 시스템 관점)
| 특징 | OTA 시스템 적용 |
|-----|---------------|
| **In-Memory Processing** | 수천 대 차량의 실시간 배포 가능 여부 판단 |
| **CDS View** | 복잡한 의존성 검증 로직을 View로 추상화 |
| **Column Store** | 버전 정보 비교 연산 최적화 |
| **High Availability** | 엔터프라이즈급 24/7 운영 안정성 |

### Database Portability
핵심 비즈니스 로직은 DB에 독립적으로 설계되었으며, 프로덕션 환경에서는:
- **SQLAlchemy ORM**을 통한 DB 추상화 계층 구현
- PostgreSQL, MySQL, Aurora 등 클라우드 DB로 전환 가능
- Connection String 변경만으로 DB 교체 지원

```python
# DB 전환 예시
# engine = create_engine('hana://user:pass@host:port')
engine = create_engine('postgresql://user:pass@host:port/dbname')
```

---

## Core Problem & Solution

### Real-World Failure Scenario
```
시나리오: BMS ECU v3.0 업데이트 시도
├─ 요구사항: BCM ECU ≥ v2.0
├─ 현재 상태: BCM ECU = v1.8
└─ 결과: ❌ Brick 발생 (차량 기능 장애)
```

### Our Solution
```
사전 검증 시스템
├─ 1단계: DependencyRule 조회
├─ 2단계: 현재 ECU 버전 확인
├─ 3단계: 버전 비교 (현재 < 최소?)
└─ 결과: ✅ READY / ❌ INCOMPATIBLE
```

> **핵심**: OTA 서버가 배포하기 전에 DB가 먼저 차단

---

## System Architecture

### High-Level Architecture
```
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│   Vehicle   │──────│  OTA Server │──────│  Dashboard  │
│  (Telematics)│      │   (API)     │      │    (Web)    │
└─────────────┘      └──────┬──────┘      └─────────────┘
                            │
                    ┌───────▼────────┐
                    │  SAP HANA DB   │
                    │                │
                    │ • Vehicles     │
                    │ • ECUs         │
                    │ • Rules        │
                    │ • Validation   │
                    └────────────────┘
```

### File Structure
```
ota-deployment-validator/
├─ scripts/
│  ├─ agent/                  # 리눅스 데이터 수집 에이전트
│  │  └─ ecu_collector.py     # /etc/ecu_info 파싱 및 데이터 전송
│  └─ populate_db.py          # 테스트 데이터 제너레이터
├─ db/                        # [HANA DB 영역]
│  ├─ src/
│  │  ├─ tables/              # CDS 기반 테이블 설계
│  │  └─ views/               # 검증 로직이 포함된 View
│
├─ backend/                   # [API 서버]
│  ├─ app.py                  # 데이터 수집 및 검증 요청 처리
│  └─ dependency_check.py     # 로직 추상화 계층
└─ README.md
```

---

## Domain Model
- 실무 OTA 환경에서는 동일 제어기라도 생산 시점에 따라 HW 버전이 다를 수 있다는 점에 착안하여, HW-SW 호환성 검증 레이어를 설계에 추가했습니다. 또한, 수백만 대의 차량을 효율적으로 관리하기 위해 '캠페인' 단위의 배치 배포 구조를 채택했습니다.

#### 핵심 도메인 모델
1. **Vehicles (차량)**: `VIN` 기반의 실차 마스터 정보 관리.
2. **Vehicle_ECU_Inventory (인벤토리)**: 각 차량의 제어기별 `HW 버전` 및 현재 `SW 버전` 실시간 추적.
3. **Update_Packages (패키지)**: 배포 파일의 보안(`Hash`) 및 목표 `HW 사양` 정의.
4. **Deployment_Campaigns (캠페인)**: 특정 모델/지역별 배포 정책 관리 (Batch 단위 배포).
5. **Dependency_Rules (의존성)**: 제어기 간 정합성 검증 규칙.
6. **Update_History (이력)**: 업데이트 과정의 상세 `Status` 및 `Error Code` 기록.
---

## Database Design

### ERD (Entity Relationship Diagram)

> OTA 업데이트 가능 여부 판단의 핵심 구조

![OTA Dependency Management Schema.png](./db.png)

### 주요 관계
- **하나의 Vehicle은 여러 ECU를 가짐 (1:N)**
  - 한 차량에는 BMS, BCM, VCU 등 여러 제어기가 탑재됨
  
- **하나의 UpdatePackage는 여러 DependencyRule을 가짐 (1:N)**
  - 하나의 패키지는 여러 선행 조건을 가질 수 있음
  - 예: BMS 3.0 업데이트는 BCM ≥ 2.0 AND VCU ≥ 1.5 요구
  
- **DependencyRule의 역할**
  - "이 패키지를 적용하려면 어떤 ECU가 어떤 버전 이상이어야 하는가"를 정의
  - Brick 방지의 핵심 로직

---

### Core Domain

실무 OTA 환경을 반영하여 **VIN(차대번호)** 중심의 자산 관리와 **캠페인 기반 배포** 구조로 설계되었습니다.

### 1. Vehicles (차량 마스터)
| Column | Type | Description |
|:--- |:--- |:--- |
| **vehicle_id** (PK) | BIGINT | 내부 식별용 고유 ID |
| **vin** (Unique) | VARCHAR(17) | 전 세계 유일 차량 식별 번호 (차대번호) |
| model_code | VARCHAR(10) | 차종 코드 (예: GN7, NQ5, EV6 등) |
| region | VARCHAR(10) | 판매 지역 (국내, 북미, 유럽 등 법규 대응용) |

### 2. Vehicle_ECU_Inventory (제어기 인벤토리)
| Column | Type | Description |
|:--- |:--- |:--- |
| vehicle_id (FK) | BIGINT | 해당 차량 연결 |
| **ecu_type** (PK) | VARCHAR(20) | 제어기 타입 (BMS, VCU, BCM 등) |
| **hw_version** | VARCHAR(20) | **물리적 HW 사양 (SW 호환성 판단의 기준)** |
| sw_major/minor/patch | INTEGER | 현재 차량에 설치된 SW 버전 정보 |
| last_reported_at | TIMESTAMP | 최종 상태 보고 시각 |

### 3. Update_Packages (OTA SW 패키지)
| Column | Type | Description |
|:--- |:--- |:--- |
| **package_id** (PK) | VARCHAR(50) | 패키지 고유 식별자 |
| **target_hw_version**| VARCHAR(20) | **설치 가능한 최소 HW 사양 (Brick 방지)** |
| **file_hash** | VARCHAR(64) | **무결성 검증용 SHA-256 해시값 (보안)** |
| sw_major/minor/patch | INTEGER | 배포될 목표 SW 버전 |

### 4. Deployment_Campaigns (배포 캠페인)
| Column | Type | Description |
|:--- |:--- |:--- |
| **campaign_id** (PK) | BIGINT | 캠페인 관리 ID |
| package_id (FK) | VARCHAR(50) | 배포할 SW 패키지 |
| target_model_code | VARCHAR(10) | 대상 차종 (예: IONIQ6 전용 배포) |
| status | VARCHAR(20) | ACTIVE, PAUSED, FINISHED |

### 5. Update_History (업데이트 이력)
| Column | Type | Description |
|:--- |:--- |:--- |
| history_id (PK) | BIGINT | 이력 식별자 |
| vin (FK) | VARCHAR(17) | 대상 차량 VIN |
| current_status | VARCHAR(20) | DOWNLOADING, INSTALLED, COMPLETED, **FAILED** |
| **error_code** | VARCHAR(10) | **실패 원인 (E104:저전압, E201:통신불량 등)** |

---

### HANA CDS View (Eligibility Check)

```sql
VIEW "OTA_SYSTEM"."v_ota_eligibility_check" AS
SELECT
  v.vin,
  v.model_code,
  c.campaign_id,
  p.package_id,
  p.target_ecu_type,
  CASE
    -- 1단계: 하드웨어 호환성 검증 (SW가 지원하는 HW 버전인가?)
    WHEN i.hw_version IS NULL THEN 'ECU_NOT_FOUND'
    WHEN i.hw_version != p.target_hw_version THEN 'INCOMPATIBLE_HW'

    -- 2단계: 의존성 제어기 존재 여부 검증 (Dependency ECU가 차량에 장착되어 있는가?)
    WHEN EXISTS (
      SELECT 1 FROM "Dependency_Rules" dr
      WHERE dr.package_id = p.package_id
        AND NOT EXISTS (
        SELECT 1 FROM "Vehicle_ECU_Inventory" e
        WHERE e.vehicle_id = v.vehicle_id
          AND e.ecu_type = dr.required_ecu_type
      )
    ) THEN 'MISSING_DEPENDENCY_ECU'

    -- 3단계: 소프트웨어 버전 의존성 검증 (선행 제어기 버전이 충분한가?)
    WHEN EXISTS (
      SELECT 1
      FROM "Dependency_Rules" dr
             INNER JOIN "Vehicle_ECU_Inventory" e
                        ON e.vehicle_id = v.vehicle_id
                          AND e.ecu_type = dr.required_ecu_type
      WHERE dr.package_id = p.package_id
        AND (
        e.sw_major_v < dr.min_sw_major_v
          OR (e.sw_major_v = dr.min_sw_major_v AND e.sw_minor_v < dr.min_sw_minor_v)
          OR (e.sw_major_v = dr.min_sw_major_v AND e.sw_minor_v = dr.min_sw_minor_v AND e.sw_patch_v < dr.min_sw_patch_v)
        )
    ) THEN 'DEPENDENCY_VERSION_MISMATCH'

    -- 4단계: 동일 버전 혹은 하위 버전 업데이트 방지 (선택 사항)
    WHEN (i.sw_major_v > p.sw_major_v)
      OR (i.sw_major_v = p.sw_major_v AND i.sw_minor_v > p.sw_minor_v)
      OR (i.sw_major_v = p.sw_major_v AND i.sw_minor_v = p.sw_minor_v AND i.sw_patch_v >= p.sw_patch_v)
      THEN 'ALREADY_UP_TO_DATE'

    ELSE 'READY'
    END AS eligibility_status,
  i.last_reported_at AS inventory_timestamp
FROM "Vehicles" v
-- 캠페인을 통해 배포 대상 차종(model_code)을 먼저 필터링 (성능 최적화 포인트)
       INNER JOIN "Deployment_Campaigns" c ON v.model_code = c.target_model_code
       INNER JOIN "Update_Packages" p ON c.package_id = p.package_id
-- 대상 제어기의 현재 상태 조인
       LEFT JOIN "Vehicle_ECU_Inventory" i ON v.vehicle_id = i.vehicle_id AND i.ecu_type = p.target_ecu_type
WHERE c.status = 'ACTIVE'                          -- 활성화된 캠페인만 조회
  AND CURRENT_TIMESTAMP BETWEEN c.start_date AND c.end_date; -- 배포 기간 내 확인
```

---

## Implementation Highlights

### 1. DB-Level Validation (Not Application-Level)
```python
# ❌ Bad: Application에서 로직 처리
for rule in dependency_rules:
    current_version = get_ecu_version(vehicle_id, rule.ecu_type)
    if current_version < rule.min_version:
        return "INCOMPATIBLE"

# ✅ Good: DB View 활용
result = session.execute(
    "SELECT eligibility_status FROM v_eligibility_check "
    "WHERE vehicle_id = ? AND package_id = ?",
    [vehicle_id, package_id]
).fetchone()
```

**장점**: 
- 데이터 정합성 보장
- 트랜잭션 안정성
- HANA In-Memory 성능 활용

### 2. Semantic Versioning Support
```python
def compare_version(current: tuple, required: tuple) -> bool:
    """
    Compare semantic versions (major, minor, patch)
    Returns True if current >= required
    """
    for curr, req in zip(current, required):
        if curr > req:
            return True
        if curr < req:
            return False
    return True  # Equal versions
```

### 3. State Machine Design
```
READY ──[start_update]──> PENDING ──[apply_package]──> UPDATING ──[success]──> READY
  │                          │                            │
  │                          │                            └──[failure]──> ERROR
  │                          │
  └──[incompatible]──────────┴────────────────────────────────────────> BLOCKED
```

---

## OTA System Relevance

본 프로젝트는 OTA 시스템에서 다음과 같은 부분 이해를 돕기 위해 진행하였습니다.

### 1. Pre-condition Validation
- 업데이트 실행 전 차량 상태 및 ECU 버전 검증
- 업데이트 불가 차량 사전 차단

### 2. Safe OTA Deployment
- ECU 간 의존성 미충족으로 인한 Brick 방지
- OTA 서버의 판단 로직 단순화

### 3. SDV Platform 확장성
- 버전 규칙 추가만으로 새로운 ECU 대응 가능
- 정책 변경 시 코드 수정 없이 DB 규칙 변경

### 4. Implementation Highlights
- 하드웨어 버전(HW Version) 무결성 보장
"소프트웨어 버전만 체크하는 것은 위험합니다. 실제 차량은 연식에 따라 제어기 HW 사양이 다르기 때문에, target_hw_version 필드를 도입하여 하드웨어-소프트웨어 간 불일치로 인한 Brick 현상을 원천 차단했습니다."

- 차분 업데이트(Delta Update) 확장성 고려
"패키지 관리 설계 시, 전체 파일 전송뿐만 아니라 버전 간 차이점만 전송하는 Delta 배포 모델로 확장할 수 있도록 패키지 메타데이터 구조를 설계했습니다."

- 상세 에러 코드를 통한 실패 분석
"단순 성공/실패 기록이 아닌, Error_Code 체계를 도입하여 배터리 전압 부족, 통신 타임아웃 등 현장의 실패 원인을 데이터화하여 배포 성공률(Success Rate) 분석이 가능케 했습니다."

### 5. Future Roadmap 
- Phase 2: 차량의 배터리 전압(SoC) 상태를 체크하는 Pre-condition 로직 추가.

- Phase 3: 대규모 차량(10만대 이상) 동시 접속 시 HANA의 Partitioning을 활용한 성능 최적화.
---

## Tech Stack

- **Database**: SAP HANA Express Edition
- **Query Language**: SQL, SQLScript, CDS View
- **API Framework**: FastAPI (Python)
- **ORM**: SQLAlchemy (DB 추상화)
- **Development Environment**: VMware (HANA Express)

---

## 🚀 Getting Started

### Prerequisites
- VMware Workstation/Fusion
- SAP HANA Express Edition (Free)
- Python 3.9+
- hdbcli

### Installation

1. **HANA DB 설치 및 실행**
```bash
# HANA Express 다운로드 및 VM 실행
# https://developers.sap.com/trials-downloads.html
```

2. **프로젝트 클론**
```bash
git clone https://github.com/Wooniq/ota-deployment-dependency-validator.git
cd ota-deployment-dependency-validator
```

3. **환경 설정**
```bash
# Python 가상환경
python -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# 의존성 설치
pip install -r requirements.txt

# 환경 변수 설정
cp backend/.env.example backend/.env
# .env 파일에서 HANA 접속 정보 입력
```

4. **DB 초기화**
```bash
# 테이블 생성 및 샘플 데이터 로드
python scripts/populate_db.py
```

5. **서버 실행**
```bash
cd backend
python app.py
```

### API Endpoints

#### 차량 배포 가능 여부 확인
```bash
GET /api/vehicles/{vehicle_id}/eligibility/{package_id}

Response:
{
  "vehicle_id": "V001",
  "package_id": "PKG_BMS_3.0",
  "status": "INCOMPATIBLE",
  "blocking_rules": [
    {
      "required_ecu": "BCM",
      "current_version": "1.8.0",
      "required_version": "2.0.0"
    }
  ]
}
```

---

## Future Roadmap

### Phase 1: Core Enhancements (완료)
- [x] HANA DB 스키마 설계
- [x] 의존성 검증 로직 구현
- [x] CDS View 기반 검증 시스템

### Phase 2: Advanced Features (계획)
- [ ] 버전 비교 로직 정규화 (Semantic Versioning)
- [ ] 대규모 Fleet Batch Validation (10,000+ 차량)
- [ ] OTA 이력 및 실패 로그 관리
- [ ] 실시간 텔레메틱스 파이프라인 연계

### Phase 3: Production Ready
- [ ] PostgreSQL/MySQL 지원 (SQLAlchemy)
- [ ] Kubernetes 배포 구성
- [ ] 모니터링 및 알림 시스템
- [ ] CI/CD 파이프라인 구축

---

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## Author

**한지운**  
현대오토에버 모빌리티 SW 스쿨 3기 (클라우드 트랙)

- GitHub: [@Wooniq](https://github.com/Wooniq)
- Project Period: 2024.12 (1주)
- Focus: SDV/OTA, Enterprise DB, Cloud Architecture

---

## References

- [SAP HANA Developer Guide](https://help.sap.com/docs/SAP_HANA_PLATFORM)
- [OTA Update Best Practices](https://www.autosar.org/)
- [Semantic Versioning 2.0.0](https://semver.org/)
