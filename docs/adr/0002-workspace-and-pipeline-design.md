# ADR 0002: MSA 워크스페이스 구조 및 Kafka 기반 데이터 파이프라인 고도화

**날짜:** 2026-02-19  
**상태:** 승인됨 (Accepted)  
**결정자:** Wooniq (System Architect)

## 1. 컨텍스트 (Context)
* 1,000대 가상 차량 시뮬레이션의 안정적 가동과 SAP HANA 2.0 SPS 08로의 실시간 데이터 적재가 필요함.
* 기존 단일 모듈 구조에서는 로컬 패키지 참조 에러(`package not in std`) 및 모듈 간 의존성 충돌 이슈가 발생함.
* 수집기(MQTT)와 저장소(HANA DB)가 직접 연결되어 있어, 트래픽 폭주 시 데이터 유실 및 DB 성능 저하 위험이 존재함.

## 2. 결정 사항 (Decisions)

### 2.1 Go Workspace 도입 및 구조 개편
* **워크스페이스 통합**: `go.work`를 통해 `cmd/agent`, `cmd/server`, `internal`을 하나의 작업 공간으로 통합 관리함.
* **로컬 참조 최적화**: `replace` 구문을 사용하여 `github.com/Wooniq/ota-agent/internal`을 로컬 디렉토리로 강제 매핑하여 원격 저장소 의존성 없이 빌드 가능하게 함.
* **네임스페이스 정규화**: 모든 임포트 경로를 `github.com/Wooniq/ota-agent/internal/...` 규격으로 통일함.

### 2.2 Kafka 3-Node 클러스터 기반 샌드위치 구조
* **디커플링(Decoupling)**: MQTT 브로커와 HANA DB 사이에 3대의 Kafka 브로커로 구성된 클러스터를 배치함.
* **가용성 및 복제**: 복제 계수(Replication Factor) 설정을 통해 특정 브로커 장애 시에도 데이터 유실을 차단함.

### 2.3 도메인 특화 상태 분석 고도화
* **상세 에러 코드 도입**: 단순 업데이트 필요 여부(`bool`)가 아닌 상세 상태 코드(`Enum`) 체계 도입.
    * `00`: 정상/성공
    * `E1`: 전압 부족 (Battery Voltage Low)
    * `E2`: 통신 타임아웃 (V2X Connection Timeout)
* **분석 엔진 통합**: Kafka Consumer 계층에서 전압/통신 상태를 분석하여 해당 코드를 부여한 뒤 DB에 최종 적재함.

## 3. 결과 (Consequences)
* **장점**:
    * 빌드 정합성 및 로컬 개발 편의성 확보.
    * 시스템 장애 허용성(Fault Tolerance) 증대로 1,000대 차량의 대규모 트래픽 안정적 처리 가능.
    * 상세 상태 코드를 활용한 관제 대시보드 분석 정밀도 향상.
* **단점**:
    * Kafka 클러스터 운영에 따른 인프라 관리 리소스 증가.
    * 메시지 직렬화/역직렬화 과정 추가에 따른 시스템 복잡도 상승.

## 4. 참고 자료
* **Database**: SAP HANA 2.0 SPS 08 (HXE)
* **Target Scale**: 1,000 Virtual Vehicles Simulation