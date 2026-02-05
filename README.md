# OTA Deployment Dependency Validation System

> **SAP HANA 기반 OTA 배포 사전 검증 시스템** > ECU 간 소프트웨어 버전 의존성을 DB 레벨에서 검증하여  
> OTA 업데이트 실패(Brick)를 사전에 방지하는 관리 플랫폼

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## Project Background

차량 OTA(Over-The-Air) 업데이트에서 가장 치명적인 문제는 **제어기(ECU) 간 버전 의존성 미검증으로 인한 업데이트 실패(Brick)** 입니다.  
본 프로젝트는 현대자동차 SDV 환경을 가정하여, 에이전트 기반 데이터 수집부터 SAP HANA DB를 통한 사전 검증까지의 End-to-End 흐름을 구현합니다.

> OTA 배포 이전에 "이 차량은 업데이트 가능한 상태인가?"를 **DB 레벨에서 명확히 판단**하는 것이 핵심입니다.

---

## System Architecture

### High-Level Architecture

```text
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

```text
ota-deployment-validator/
├─ scripts/
│  ├─ agent/                  # Go 기반 데이터 수집 에이전트
│  │  └─ ecu_collector.go     # /etc/ecu_info 파싱 및 데이터 전송
│  └─ populate_db.py          # 테스트 데이터 제너레이터
├─ db/                        # [HANA DB 영역]
│  ├─ src/
│  │  ├─ tables/              # CDS 기반 테이블 설계
│  │  └─ views/               # 검증 로직이 포함된 View
├─ backend/                   # [API 서버]
│  ├─ app.py                  # 데이터 수집 및 검증 요청 처리
│  └─ dependency_check.py     # 로직 추상화 계층
└─ README.md
```

---

## Domain Model

> 실무 OTA 환경에서는 동일 제어기라도 생산 시점에 따라 HW 버전이 다를 수 있다는 점에 착안하여, <br> HW-SW 호환성 검증 레이어를 설계에 추가했습니다. <br>
> 또한, 수백만 대의 차량을 효율적으로 관리하기 위해 '캠페인' 단위의 배치 배포 구조를 채택했습니다.

1. **Vehicles**: `VIN` 기반의 실차 마스터 정보 관리

2. **Vehicle_ECU_Inventory**: 각 차량의 제어기별 `HW 버전` 및 현재 `SW 버전` 실시간 추적

3. **Update_Packages**: 배포 파일의 보안(`Hash`) 및 목표 `HW 사양` 정의

4. **Dependency_Rules)**: 제어기 간 정합성 검증 규칙 (예: BMS 업데이트 시 BCM 버전 체크)

6. **Update_History**: 업데이트 과정의 상세 `Status` 및 `Error Code` 기록

---

## Database Design

### ERD (Entity Relationship Diagram)

> OTA 업데이트 가능 여부 판단의 핵심 구조

![OTA Dependency Management Schema.png](./db.png)

---

## Implementation Highlights
- **HW Version 무결성 보장**: `target_hw_version` 필드를 도입하여 하드웨어-소프트웨어 불일치로 인한 Brick 현상을 원천 차단했습니다.

- **상세 에러 코드 체계**: 단순 실패 기록이 아닌 전압 부족, 통신 타임아웃 등 현장의 실패 원인을 데이터화하여 배포 성공률 분석이 가능케 했습니다.

- **DB-Level Validation**: 복잡한 의존성 체크 로직을 애플리케이션이 아닌 SAP HANA의 **CDS View**에서 처리하여 데이터 정합성과 고속 조인 성능을 확보했습니다.

---

## Tech Stack

- **Database**: SAP HANA Express Edition (CDS View, SQLScript)

- **Backend**: FastAPI (Python 3.9+), SQLAlchemy

- **Agent**: Go (ECU Version Collector)

- **Environment**: VMware (HANA Express VM)

- **Development Environment**: VMware (HANA Express)

---

## 🚀 Getting Started

**Installation & Run**

1. **HANA DB 실행**: VM 환경에서 HANA Express Edition 기동

2. **프로젝트 클론**: `git clone https://github.com/Wooniq/ota-deployment-dependency-validator.git`

3. **환경 설정**: `.env` 파일에 DB 접속 정보 입력

4. **DB 초기화**: `python scripts/populate_db.py` (테이블 생성 및 샘플 데이터 로드)

5. 서버 실행: `cd backend && python app.py`


---

## Author

**한지운** (현대오토에버 모빌리티 SW 스쿨 3기)

- Focus: SDV/OTA, Enterprise DB, Cloud Architecture

- GitHub: [@Wooniq](https://github.com/Wooniq)

---

## References

- [SAP HANA Developer Guide](https://help.sap.com/docs/SAP_HANA_PLATFORM)

- [OTA Update Best Practices](https://www.autosar.org/)

- [Semantic Versioning 2.0.0](https://semver.org/)