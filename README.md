# OTA Deployment Dependency Validation System (SAP HANA)

> **SAP HANA 기반 OTA 배포 사전 검증 시스템**  
> ECU 간 소프트웨어 버전 의존성을 DB 레벨에서 검증하여  
> OTA 업데이트 실패(Brick)를 사전에 방지하는 관리 플랫폼

---

## Project Background

차량 OTA(Over-The-Air) 업데이트에서 가장 치명적인 문제는  
**제어기(ECU) 간 버전 의존성 미검증으로 인한 업데이트 실패(Brick)** 입니다.

본 프로젝트는 현대자동차 SDV / OTA 환경을 가정하여, **SAP HANA DB 레벨에서 배포 전 사전 검증**을 수행하는 시스템을 구현합니다.

### Key Objectives
- ECU 간 **SW 버전 의존성 모델링**
- OTA 패키지 배포 가능 여부를 **DB에서 사전 판단**
- Brick 방지를 위한 **Pre-condition Check** 구조 설계
- SAP HANA의 **In-Memory 고속 조인** 활용

> OTA 배포 이전에 “이 차량은 업데이트 가능한 상태인가?”를  
> **Yes / No로 명확히 판단**하는 것이 핵심입니다.

---

## Why SAP NANA?
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

- **Vehicle** : 차량 단위 정보
- **ECU** : 차량 내 제어기 및 현재 SW 버전
- **UpdatePackage** : OTA 업데이트 대상 패키지
- **DependencyRule** : ECU 간 버전 의존성 규칙

---

## ERD (Entity Relationship Diagram)

> OTA 업데이트 가능 여부 판단의 핵심 구조

![ERD](./docs/erd.png)

### 주요 관계
- 하나의 Vehicle은 여러 ECU를 가짐 (1:N)
- 하나의 UpdatePackage는 여러 DependencyRule을 가짐 (1:N)
- DependencyRule은 “이 패키지를 적용하려면 어떤 ECU가 어떤 버전 이상이어야 하는가”를 정의

---

## Database Schema

### Vehicles (차량 마스터)
| Column | Type | Description |
|:--- |:--- |:--- |
| **vehicle_id** (PK) | VARCHAR | 차량 고유 식별자 (VIN 등) |
| model | VARCHAR | 차종 (예: IONIQ6, GV80) |
| status | VARCHAR | 차량 상태 (READY / UPDATING / ERROR) |

### ECUs (제어기 현황)
| Column | Type | Description |
|:--- |:--- |:--- |
| id (PK) | INTEGER | 식별자 (Auto Increment) |
| vehicle_id (FK) | VARCHAR | 소속 차량 ID |
| ecu_type | VARCHAR | 제어기 타입 (BMS, BCM, VCU 등) |
| **major_v** | INTEGER | 현재 SW 주 버전 |
| **minor_v** | INTEGER | 현재 SW 부 버전 |
| **patch_v** | INTEGER | 현재 SW 패치 버전 |

### UpdatePackages (신규 배포 패키지)
| Column | Type | Description |
|:--- |:--- |:--- |
| **package_id** (PK) | VARCHAR | OTA 패키지 ID |
| target_ecu_type | VARCHAR | 업데이트 대상 제어기 타입 |
| target_major_v | INTEGER | 목표 주 버전 |
| target_minor_v | INTEGER | 목표 부 버전 |
| target_patch_v | INTEGER | 목표 패치 버전 |

### DependencyRules (의존성 검증 규칙)
| Column | Type | Description |
|:--- |:--- |:--- |
| rule_id (PK) | INTEGER | 규칙 고유 ID |
| package_id (FK) | VARCHAR | 연결된 OTA 패키지 ID |
| required_ecu_type | VARCHAR | 선행 조건을 확인할 제어기 타입 |
| **min_major_v** | INTEGER | 요구되는 최소 주 버전 |
| **min_minor_v** | INTEGER | 요구되는 최소 부 버전 |
| **min_patch_v** | INTEGER | 요구되는 최소 패치 버전 |

---

## Core Logic: Update Eligibility Check

### Question
> “이 차량은 이 OTA 업데이트를 적용할 수 있는가?”

### Decision Flow
1. OTA 패키지에 정의된 DependencyRule 조회
2. 해당 차량의 ECU 현재 버전 조회
3. **현재 버전 < 최소 요구 버전**
    - Yes → ❌ INCOMPATIBLE
    - No → ✅ READY

### Result Example

| vehicle_id | package_id | eligibility |
|-----------|------------|-------------|
| V001 | PKG_001 | INCOMPATIBLE |

---

## Implementation Highlights

- **SAP HANA**
    - In-memory 기반 고속 조인
    - SQLScript / CDS View 활용
- **Validation Logic**
    - Application 레벨이 아닌 **DB 레벨 판단**
    - 데이터 정합성 중심 설계
- **State Management**
    - READY → PENDING → UPDATING
    - INCOMPATIBLE → BLOCKED

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

- **Database**: SAP HANA
- **Query / Logic**: SQL, SQLScript, CDS View
- **Optional API**: REST API or Stored Procedure
- **Visualization**: SAPUI5 / SQL Console

---

## Future Improvements

- 버전 비교 로직 정규화 (Semantic Versioning)
- 대규모 차량 Fleet 대상 Batch Validation
- OTA 이력 및 실패 로그 관리
- 실시간 텔레메틱스 파이프라인과 연계

---

## Author

- **Name**: 한지운
- **Program**: 현대오토에버 모빌리티 SW 스쿨 3기 (클라우드 트랙)
