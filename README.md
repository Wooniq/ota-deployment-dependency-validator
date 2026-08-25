# OTA Deployment Dependency Validator

> 차량 OTA 업데이트 전 ECU 간 소프트웨어 의존성을 사전 검증하여 업데이트 실패(Brick)를 방지하는 관제 시스템

[![Java](https://img.shields.io/badge/Java-17-orange)](https://adoptium.net/)
[![Spring Boot](https://img.shields.io/badge/Spring%20Boot-3.3.5-green)](https://spring.io/projects/spring-boot)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8)](https://go.dev/)
[![SAP HANA](https://img.shields.io/badge/SAP%20HANA-Express-0FAAFF)](https://www.sap.com/products/technology-platform/hana.html)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

---

## 프로젝트 배경

차량 OTA(Over-The-Air) 업데이트에서 가장 치명적인 문제는 **제어기(ECU) 간 버전 의존성 미검증으로 인한 업데이트 실패(Brick)** 입니다.

예를 들어, BMS(배터리 관리 시스템)를 v3.0으로 업데이트하려면 BCM(차체 제어 모듈)이 최소 v1.2.0 이상이어야 합니다. 이 의존성을 무시하고 배포하면 차량이 벽돌(Brick) 상태에 빠질 수 있습니다.

본 시스템은 현대자동차 SDV 환경을 가정하여, **Go 에이전트 기반 데이터 수집 → Kafka/MQTT 메시징 → Spring Boot 관제 서버 → SAP HANA DB 검증**까지의 End-to-End 흐름을 구현합니다.

---

## 시스템 아키텍처

```
                    ┌──────────────────────┐
                    │   Backoffice (React)  │
                    │   S3 Static Hosting   │
                    └──────────┬───────────┘
                               │ REST API
                    ┌──────────▼───────────┐
                    │     nginx Ingress     │
                    │     (NodePort 30080)  │
                    └──────────┬───────────┘
┌─────────────┐                │            ┌─────────────────────┐
│   Vehicle    │    ┌──────────▼──────────┐ │   Kafka 3-Node      │
│   Agent (Go) │───▶│  OTA Control Server │◀│   (Event Bus)       │
│              │    │  (Spring Boot 3.x)  │▶│                     │
│ • DLT Parser │    │                     │ └─────────────────────┘
│ • VIN Valid. │    │ • Validator Service  │
│ • MQTT Pub   │    │ • Analyzer Service  │
└──────┬───────┘    │ • Campaign Service  │
                    │ • Firmware Service  │
       │            └──────────┬──────────┘
       │ MQTT                  │ JDBC
       ▼                       ▼
┌─────────────┐    ┌─────────────────────┐
│ MQTT Broker │    │     SAP HANA DB     │
│ (Mosquitto) │    │                     │
└─────────────┘    │ • Vehicles          │
                   │ • ECU Inventory     │
                   │ • Update Packages   │
                   │ • Dependency Rules  │
                   │ • Campaigns         │
                   │ • Update History    │
                   └─────────────────────┘
```

### 데이터 흐름

1. **수집**: Go 에이전트가 차량 ECU 상태를 AUTOSAR DLT 프로토콜로 수집, MQTT로 발행
2. **적재**: MQTT Collector가 메시지를 Kafka 토픽(`ota-inventory`)에 적재
3. **분석**: Kafka Consumer가 인벤토리 데이터를 분석, 기준 버전 미달 시 롤백 명령 발송
4. **검증**: 업데이트 배포 전 의존성 규칙을 DB 레벨에서 사전 검증
5. **배포**: 캠페인 단위로 대상 차량 그룹에 펌웨어 배포

---

## 프로젝트 구조

```
ota-deployment-dependency-validator/
│
├── agent/                              # Go 차량 에이전트
│   └── gen_vehicles.py                 # 차량 시뮬레이터
│
├── cmd/                                # Go 에이전트 엔트리포인트
│   └── server/
│       └── main.go                     # HANA + Kafka + MQTT 부트스트랩
│
├── pkg/                                # Go 공유 패키지
│   ├── repository/                     # HANA DB 접근 계층
│   ├── service/                        # OTA 분석 엔진
│   └── transport/                      # Kafka Producer/Consumer, MQTT Client
│
├── server/                             # Spring Boot 관제 서버
│   ├── src/main/java/com/ota/control/
│   │   ├── domain/                     # JPA Entity
│   │   │   ├── Vehicle.java            # 차량 정보 + 상태 관리
│   │   │   ├── Ecu.java                # ECU 타입별 버전 (SemVer 비교 내장)
│   │   │   ├── UpdatePackage.java      # 펌웨어 패키지 + S3 키
│   │   │   ├── DependencyRule.java     # ECU 간 버전 의존성 규칙
│   │   │   └── Campaign.java           # 배포 캠페인 상태 머신
│   │   │
│   │   ├── service/                    # 핵심 비즈니스 로직
│   │   │   ├── OtaValidatorService.java    # 의존성 검증 엔진
│   │   │   ├── OtaAnalyzerService.java     # 인벤토리 분석 + 롤백 판단
│   │   │   └── S3FirmwareService.java      # Presigned URL 발급
│   │   │
│   │   ├── controller/                 # REST API
│   │   │   ├── VehicleController.java      # 차량/ECU CRUD
│   │   │   ├── OtaCheckController.java     # /check-update 검증 API
│   │   │   ├── FirmwareController.java     # S3 업/다운로드 URL
│   │   │   └── CampaignController.java     # 캠페인 생명주기 관리
│   │   │
│   │   ├── messaging/                  # 메시징 계층
│   │   │   ├── KafkaInventoryProducer.java # Kafka 전송
│   │   │   ├── KafkaInventoryConsumer.java # Kafka 수신 → Analyzer
│   │   │   ├── MqttCollector.java          # MQTT → Kafka 브릿지
│   │   │   └── MqttPublisher.java          # 롤백 커맨드 발송
│   │   │
│   │   └── config/                     # 설정
│   │       ├── MqttConfig.java
│   │       ├── S3Config.java
│   │       └── DataInitializer.java        # 샘플 데이터 (local 전용)
│   │
│   ├── src/main/resources/
│   │   └── application.yml             # H2(local) / HANA(prod) 프로필
│   │
│   ├── src/test/                       # 단위 테스트
│   ├── Dockerfile                      # 멀티스테이지 빌드
│   ├── build.gradle
│   └── docker-compose.yml              # 전체 스택 (서버+Kafka+MQTT)
│
├── k8s/                                # Kubernetes 매니페스트
│   ├── ota-server.yaml                 # Deployment + ClusterIP
│   └── nginx.yaml                      # nginx 리버스프록시 + NodePort
│
├── db/                                 # HANA DB 스키마 (CDS View)
├── docs/                               # 문서
├── infra/                              # 인프라 설정
├── docker-compose.yml                  # 루트 레벨 통합 compose
└── README.md
```

---

## 데이터베이스 설계

### ERD (Entity Relationship Diagram)

![ERD](./db.png)

### 핵심 테이블

| 테이블 | 역할 | 설계 포인트 |
|---|---|---|
| **Vehicles** | VIN 기반 실차 마스터 정보 | ISO 3779 VIN 검증 (I, O, Q 미사용) |
| **Vehicle_ECU_Inventory** | 차량별 ECU HW/SW 버전 실시간 추적 | Composite PK (vehicle_id + ecu_type) |
| **Update_Packages** | 펌웨어 패키지 + HW 호환성 정의 | `target_hw_version`으로 HW-SW 불일치 차단 |
| **Dependency_Rules** | ECU 간 버전 의존성 규칙 | Semantic Versioning 기반 비교 |
| **Deployment_Campaigns** | 배치 배포 단위 관리 | 상태 머신 (CREATED → IN_PROGRESS → COMPLETED) |
| **Update_History** | 업데이트 이력 + 실패 원인 분석 | 전압 부족, 통신 타임아웃 등 에러 코드 체계 |

---

## 핵심 기능

### 1. OTA 의존성 사전 검증

차량의 현재 ECU 버전과 업데이트 패키지의 의존성 규칙을 비교하여 배포 가능 여부를 판별합니다.

```
POST /api/ota/check-update
{
  "vehicleId": "V001",
  "packageId": "PKG_BMS_30"
}

→ 200 OK
{
  "vehicleId": "V001",
  "packageId": "PKG_BMS_30",
  "available": true,
  "details": [
    {
      "rule": "BCM",
      "status": "PASS",
      "currentVersion": "1.5.0",
      "requiredVersion": "1.2.0"
    }
  ]
}
```

### 2. 실시간 인벤토리 분석 + 자동 롤백

Kafka를 통해 수집된 차량 ECU 데이터를 실시간 분석하고, 기준 버전 미달 시 MQTT를 통해 차량에 롤백 명령을 자동 발송합니다.

```
Vehicle Agent ──MQTT──▶ Collector ──Kafka──▶ Analyzer ──MQTT──▶ Rollback Command
                                                │
                                          기준 미달 감지 시
                                          차량 상태 → ROLLBACK
```

### 3. 캠페인 기반 배포 관리

(설계 목표) 수백만 대 규모의 차량군까지 확장 가능하도록, 캠페인 단위의 배치 배포 구조를 채택했습니다. 현재는 1,000대 시뮬레이션 기준으로 검증되었습니다.

```
CREATED → VALIDATING → IN_PROGRESS → COMPLETED
                                   → PAUSED
                                   → ABORTED
```

### 4. S3 펌웨어 파일 관리

Presigned URL 기반으로 프론트엔드에서 직접 S3에 업로드/다운로드하여 서버 부하를 최소화합니다.

---

## API 명세

| Method | Endpoint | 설명 |
|---|---|---|
| `GET` | `/api/ota` | 서버 헬스체크 |
| `POST` | `/api/ota/check-update` | OTA 의존성 검증 |
| `GET` | `/api/vehicles` | 전체 차량 조회 |
| `POST` | `/api/vehicles` | 차량 등록 |
| `GET` | `/api/vehicles/{id}/ecus` | 차량 ECU 목록 |
| `POST` | `/api/vehicles/{id}/ecus` | ECU 등록 |
| `POST` | `/api/firmware/upload-url` | 펌웨어 업로드 URL 발급 |
| `GET` | `/api/firmware/download-url` | 펌웨어 다운로드 URL 발급 |
| `GET` | `/api/campaigns` | 캠페인 목록 |
| `POST` | `/api/campaigns` | 캠페인 생성 |
| `PUT` | `/api/campaigns/{id}/start` | 캠페인 시작 |
| `PUT` | `/api/campaigns/{id}/abort` | 캠페인 중단 |

---

## 기술 스택

### Backend
| 구분 | 기술 | 용도 |
|---|---|---|
| Language | Java 17 | 관제 서버 |
| Framework | Spring Boot 3.3.5 | REST API, DI, JPA |
| ORM | Spring Data JPA + Hibernate | HANA DB 접근 |
| Messaging | Spring Kafka | 이벤트 기반 데이터 파이프라인 |
| MQTT | Eclipse Paho | 차량 ↔ 서버 양방향 통신 |
| Storage | AWS S3 (Presigned URL) | 펌웨어 파일 관리 |

### Agent
| 구분 | 기술 | 용도 |
|---|---|---|
| Language | Go 1.25 | 경량 차량 에이전트 |
| Protocol | MQTT, Kafka | 텔레메트리 발행, 이벤트 전송 |
| DB Driver | SAP go-hdb | HANA DB 직접 접근 |

### Infrastructure
| 구분 | 기술 | 용도 |
|---|---|---|
| Database | SAP HANA Express | 메인 DB (CDS View 기반 검증) |
| Message Broker | Apache Kafka 3.7 (KRaft) | 3-node 클러스터, 이벤트 버스 |
| MQTT Broker | Eclipse Mosquitto | 차량 텔레메트리 수집 |
| Container | Docker, Kubernetes | 컨테이너 오케스트레이션 |
| Reverse Proxy | nginx | K8s Ingress, NodePort 30080 |
| Cloud | AWS (EC2, S3, RDS) | 서버 호스팅, 파일 스토리지 |

---

## 실행 가이드

### 로컬 개발 (H2 인메모리)

```bash
cd server
./gradlew bootRun
# H2 DB + 샘플 데이터 자동 로드
# http://localhost:8080/api/vehicles
# http://localhost:8080/h2-console (JDBC URL: jdbc:h2:mem:otadb)
```

### Docker (전체 인프라)

```bash
cp .env.example .env
# .env에 HANA 접속 정보 입력
docker compose up -d
```

### Kubernetes 배포

```bash
# 이미지 빌드
docker build -t ota-control-server:latest server/

# DB Secret 생성
kubectl create secret generic ota-db-secret \
  --from-literal=address=<HANA_HOST> \
  --from-literal=port=30015 \
  --from-literal=username=DBADMIN \
  --from-literal=password=<PASSWORD>

# 배포
kubectl apply -f k8s/ota-server.yaml
kubectl apply -f k8s/nginx.yaml

# 접속 확인
curl http://<EC2_PUBLIC_IP>:30080/api/ota
```

---

## 설계 결정 사항

| 결정 | 선택 | 근거 |
|---|---|---|
| 관제 서버 언어 | Java/Spring Boot | 엔터프라이즈 생태계 (Security, JPA, Kafka Starter), 현대오토에버 실무 표준 |
| 차량 에이전트 언어 | Go | 경량 바이너리, 낮은 리소스 사용, 임베디드 환경 적합 |
| 에이전트 ↔ 서버 통신 | MQTT + Kafka | MQTT: 경량 텔레메트리, Kafka: 안정적 이벤트 파이프라인 |
| DB 검증 전략 | DB-Level Validation | CDS View에서 의존성 체크 → 데이터 정합성 + 고속 조인 |
| 배포 단위 | 캠페인 기반 | (설계 목표) 수백만 대 규모까지 확장 가능한 배치 관리 구조. 현재 검증 규모는 1,000대 |
| HW 호환성 | `target_hw_version` 필드 | 동일 ECU라도 생산 시점별 HW 차이 → Brick 원천 차단 |

---

## 향후 계획

- [ ] gRPC 프로토콜 도입 (에이전트 ↔ 서버 명령 채널)
- [ ] AUTOSAR DLT 빅엔디안 파서 고도화
- [ ] ISO 3779 VIN 검증기 통합
- [ ] Redis 캐시 레이어 (차량 세션, 캠페인 상태)
- [ ] React 백오피스 대시보드
- [ ] MSA 서비스 분리 (inventory / validator / campaign / firmware / notification)
- [ ] Prometheus + Grafana 모니터링

---

## Author

**한지운** — 현대오토에버 모빌리티 SW 스쿨 3기

- Focus: SDV/OTA, Enterprise DB, Cloud Architecture
- GitHub: [@Wooniq](https://github.com/Wooniq)