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



```

┌─────────────┐ ┌─────────────┐ ┌─────────────┐

│ Vehicle │──────│ OTA Server │──────│ Dashboard │

│ (Telematics)│ │ (API) │ │ (Web) │

└─────────────┘ └──────┬──────┘ └─────────────┘

│

┌───────▼────────┐

│ SAP HANA DB │

│ │

│ • Vehicles │

│ • ECUs │

│ • Rules │

│ • Validation │

└────────────────┘

```



### File Structure

```

ota-deployment-validator/

├─ scripts/

│ ├─ agent/ # 리눅스 데이터 수집 에이전트

│ │ └─ ecu_collector.py # /etc/ecu_info 파싱 및 데이터 전송

│ └─ populate_db.py # 테스트 데이터 제너레이터

├─ db/ # [HANA DB 영역]

│ ├─ src/

│ │ ├─ tables/ # CDS 기반 테이블 설계

│ │ └─ views/ # 검증 로직이 포함된 View

│

├─ backend/ # [API 서버]

│ ├─ app.py # 데이터 수집 및 검증 요청 처리

│ └─ dependency_check.py # 로직 추상화 계층

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



### State Machine Design

```

READY ──[start_update]──> PENDING ──[apply_package]──> UPDATING ──[success]──> READY

│ │ │

│ │ └──[failure]──> ERROR

│ │

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

source venv/bin/activate # Windows: venv\Scripts\activate



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