# OTA Deployment Dependency Validation System

> **SAP HANA 기반 OTA 배포 사전 검증 시스템**  
> ECU 간 소프트웨어 버전 의존성을 DB 레벨에서 검증하여  
> OTA 업데이트 실패(Brick)를 사전에 방지하는 관리 플랫폼

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Project Background

차량 OTA(Over-The-Air) 업데이트에서 가장 치명적인 문제는  
**제어기(ECU) 간 버전 의존성 미검증으로 인한 업데이트 실패(Brick)** 입니다.

본 프로젝트는 현대자동차 SDV / OTA 환경을 가정하여, **SAP HANA DB 레벨에서 배포 전 사전 검증**을 수행하는 시스템을 구현합니다.

### Key Objectives
- ✅ ECU 간 **SW 버전 의존성 모델링**
- ✅ OTA 패키지 배포 가능 여부를 **DB에서 사전 판단**
- ✅ Brick 방지를 위한 **Pre-condition Check** 구조 설계
- ✅ SAP HANA의 **In-Memory 고속 조인** 활용

> OTA 배포 이전에 "이 차량은 업데이트 가능한 상태인가?"를  
> **Yes / No로 명확히 판단**하는 것이 핵심입니다.

---

## 🔍 Why SAP HANA?

### 현대자동차 기술 스택 경험
본 프로젝트는 현대자동차그룹의 실제 엔터프라이즈 환경을 이해하기 위해 SAP HANA DB를 선택했습니다.

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
├─ db/                        # [HANA DB 영역]
│  ├─ src/
│  │  ├─ tables/
│  │  │  └─ ota_tables.hdbcds # 테이블 정의 (CDS)
│  │  ├─ views/
│  │  │  └─ v_eligibility.hdbview # 검증 View
│  │  └─ data/
│  │     └─ sample_data.csv   # 초기 데이터
│  └─ schema.dbml             # ERD 설계
│
├─ backend/                   # [API 서버]
│  ├─ .env.example            # HANA 접속 정보
│  ├─ app.py                  # FastAPI 진입점
│  ├─ database.py             # DB 연결 관리
│  ├─ dependency_check.py     # 핵심 검증 로직
│  └─ models.py               # Pydantic 모델
│
├─ frontend/                  # [대시보드]
│  └─ dashboard/              # React 기반 UI
│
└─ scripts/                   # [유틸리티]
   └─ populate_db.py          # 데이터 초기화
```

---

## Domain Model

본 프로젝트는 아래 4개의 핵심 도메인으로 구성됩니다.

- **Vehicle**: 차량 단위 정보
- **ECU**: 차량 내 제어기 및 현재 SW 버전
- **UpdatePackage**: OTA 업데이트 대상 패키지
- **DependencyRule**: ECU 간 버전 의존성 규칙

---

## Database Design

### ERD (Entity Relationship Diagram)

> OTA 업데이트 가능 여부 판단의 핵심 구조

<img width="872" height="654" alt="OTA Dependency Management Schema" src="https://github.com/user-attachments/assets/49d541ff-f8d4-43e1-8ea5-c6994b0002f3" />

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

### Core Tables

#### Vehicles (차량 마스터)
| Column | Type | Description |
|:--- |:--- |:--- |
| **vehicle_id** (PK) | VARCHAR(50) | 차량 고유 식별자 (VIN) |
| model | VARCHAR(50) | 차종 (IONIQ6, GV80, EV6 등) |
| status | VARCHAR(20) | 차량 상태 (READY/UPDATING/ERROR/BLOCKED) |
| created_at | TIMESTAMP | 최초 등록 시간 |
| updated_at | TIMESTAMP | 상태 변경 시간 |

**상태 값 (status)**:
- `READY`: OTA 배포 가능 상태
- `UPDATING`: 업데이트 진행 중 (이중 배포 차단)
- `ERROR`: 업데이트 실패 (점검 필요)
- `BLOCKED`: 의존성 미충족 (배포 불가)

---

#### ECUs (제어기 현황)
| Column | Type | Description |
|:--- |:--- |:--- |
| id (PK) | INTEGER | 식별자 (Auto Increment) |
| vehicle_id (FK) | VARCHAR(50) | 소속 차량 ID |
| ecu_type | VARCHAR(20) | 제어기 타입 |
| **major_v** | INTEGER | 현재 SW 주 버전 |
| **minor_v** | INTEGER | 현재 SW 부 버전 |
| **patch_v** | INTEGER | 현재 SW 패치 버전 |

**제어기 종류 (ecu_type)**:
- `BMS` (Battery Management System): 배터리 관리
- `BCM` (Body Control Module): 바디 제어
- `VCU` (Vehicle Control Unit): 차량 통합 제어
- `ADAS` (Advanced Driver Assistance): 자율주행 보조
- `Gateway`: 차량 내 네트워크 게이트웨이

**인덱스**:
```sql
-- 차량당 ECU 타입 중복 방지 + 고속 조회
UNIQUE INDEX (vehicle_id, ecu_type);
```

---

#### UpdatePackages (신규 배포 패키지)
| Column | Type | Description |
|:--- |:--- |:--- |
| **package_id** (PK) | VARCHAR(50) | OTA 패키지 ID (예: PKG_BMS_3.0.0) |
| target_ecu_type | VARCHAR(20) | 업데이트 대상 제어기 타입 |
| target_major_v | INTEGER | 목표 주 버전 |
| target_minor_v | INTEGER | 목표 부 버전 |
| target_patch_v | INTEGER | 목표 패치 버전 |

---

#### DependencyRules (의존성 검증 규칙)
| Column | Type | Description |
|:--- |:--- |:--- |
| rule_id (PK) | INTEGER | 규칙 고유 ID (Auto Increment) |
| package_id (FK) | VARCHAR(50) | 연결된 OTA 패키지 ID |
| required_ecu_type | VARCHAR(20) | 선행 조건을 확인할 제어기 타입 |
| **min_major_v** | INTEGER | 요구되는 최소 주 버전 |
| **min_minor_v** | INTEGER | 요구되는 최소 부 버전 |
| **min_patch_v** | INTEGER | 요구되는 최소 패치 버전 |

**의존성 예시**:
| Package | Required ECU | Min Version | Reason |
|:--- |:--- |:--- |:--- |
| PKG_BMS_3.0.0 | BCM | 2.0.0 | CAN 통신 프로토콜 변경 |
| PKG_VCU_2.5.0 | Gateway | 1.8.0 | 새로운 메시지 타입 사용 |

---

### HANA CDS View (Eligibility Check)

```sql
VIEW v_eligibility_check AS
SELECT 
    v.vehicle_id,
    p.package_id,
    CASE 
        -- Rule이 없으면 무조건 가능
        WHEN NOT EXISTS (
            SELECT 1 FROM dependency_rules dr
            WHERE dr.package_id = p.package_id
        ) THEN 'READY'
        
        -- 필요한 ECU가 없으면 차단
        WHEN EXISTS (
            SELECT 1 FROM dependency_rules dr
            WHERE dr.package_id = p.package_id
            AND NOT EXISTS (
                SELECT 1 FROM ecus e
                WHERE e.vehicle_id = v.vehicle_id
                AND e.ecu_type = dr.required_ecu_type
            )
        ) THEN 'MISSING_ECU'
        
        -- 버전 미달이면 차단
        WHEN EXISTS (
            SELECT 1 
            FROM dependency_rules dr
            INNER JOIN ecus e 
                ON e.vehicle_id = v.vehicle_id
                AND e.ecu_type = dr.required_ecu_type
            WHERE dr.package_id = p.package_id
                AND (
                    e.major_v < dr.min_major_v 
                    OR (e.major_v = dr.min_major_v AND e.minor_v < dr.min_minor_v)
                    OR (e.major_v = dr.min_major_v AND e.minor_v = dr.min_minor_v AND e.patch_v < dr.min_patch_v)
                )
        ) THEN 'INCOMPATIBLE'
        
        ELSE 'READY'
    END AS eligibility_status
FROM vehicles v
CROSS JOIN update_packages p;
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
