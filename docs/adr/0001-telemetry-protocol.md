# ADR 0001: 차량 텔레메트리 및 데이터 파이프라인 프로토콜 선정

## 1. 상태 (Status)
제안됨 (Proposed) - 2026-02-13

## 2. 배경 (Context)
1,000대 이상의 가상 차량 에이전트가 생성하는 DLT(Diagnostic Log and Trace) 바이너리 데이터를 실시간으로 수집하고 검증해야 함. 대규모 트래픽 환경에서 시스템 안정성과 데이터 정합성을 확보하기 위한 최적의 통신 스택 선정이 필요함.

## 3. 결정 (Decision)
차량-서버 구간(Edge-to-Cloud)은 **MQTT**를 채택하고, 서버 내부 파이프라인(Internal)은 **Kafka**를 배치하는 하이브리드 아키텍처를 구성함.



### 3.1 MQTT 선정의 타당성 (Vehicle-to-Cloud)
* **경량화 및 효율성**: HTTP 대비 오버헤드가 적어 데이터 전송 효율을 약 30% 개선 가능. 대규모 에이전트 환경에서 전송 비용 절감 및 속도 최적화.
* **네트워크 안정성**: 이동성(Mobility) 환경 특유의 불안정한 네트워크 상황에서 MQTT의 `Keep Alive` 및 `Auto-Reconnect` 메커니즘을 통해 "끊김 없는 업데이트" 환경 구현.
* **실시간성(Low Latency)**: Pub/Sub 구조를 통해 수천 개의 로그가 동시 발생할 때 저지연 통신을 보장하여 데이터 가용성 극대화.

### 3.2 Kafka 선정의 타당성 (Internal Data Pipeline)
* **트래픽 버퍼링**: 대량의 에이전트 데이터를 DB(SAP HANA 등)에 직접 쓰기 시 발생하는 병목 현상을 방지하기 위해 Kafka를 중간 버퍼로 배치.
* **고가용성 및 확장성**: 모빌리티 클라우드 기조에 맞춰 수만 대 이상의 디바이스 확장을 수용할 수 있는 부하 분산 구조 확보.
* **분석 연동성**: 수집된 DLT 로그를 실시간 분석하거나 ELK 스택 등 외부 로그 분석 파이프라인으로 확장하기 용이함.

## 4. 기대 효과 (Consequences)
* **도메인 정합성**: "업데이트 실패(Brick) 방지를 위한 정교한 검증 모델"이라는 프로젝트 핵심 가치를 기술적으로 뒷받침.
* **기술적 차별화**: 차량 도메인의 특수성(네트워크 불안정)과 서버 가용성(DB 쓰기 병목)을 동시에 해결한 실무 중심적 설계 입증.


## 5. 레퍼런스 및 기술 근거 (References & Alignment)

본 의사결정은 현대자동차그룹의 SDV(Software Defined Vehicle) 가속화 전략 및 현대오토에버의 모빌리티 클라우드 아키텍처 가이드라인을 준수합니다.

### 5.1 공식 기술 포털
* **[HMG Developer Center]** [SDV를 향한 소프트웨어 아키텍처의 변화](https://developers.hyundaimotorgroup.com/): 도메인 컨트롤러와 클라우드 간의 유기적 데이터 연결 아키텍처 참고.
* **[Hyundai AutoEver Tech Blog]** [모빌리티 데이터 플랫폼 구축 전략](https://tech.hyundai-autoever.com/): 대규모 차량 텔레메트리 수집을 위한 Kafka 기반 Ingestion 파이프라인 설계 근거.

### 5.2 산업 표준 규격
* **[AUTOSAR Standard]** [Diagnostic Log and Trace (DLT)](https://www.autosar.org/): 차량 내부 제어기 로그 표준 규격 준수 및 바이너리 직렬화 로직 설계.
* **[ISO 3779]** [Road vehicles — VIN](https://www.iso.org/standard/43111.html): `gen_vehicles.py` 내 차대번호 생성 로직의 국제 표준 정합성 확보.

### 5.3 전략 발표 자료
* **[Hyundai Motor Group]** [2022 SDV UNVEIL 전략 발표](https://www.hyundai.co.kr/): 2025년 전 차종 SDV 전환에 따른 '데이터 가용성' 및 '무선 업데이트(OTA)'의 기술적 중요성 반영.