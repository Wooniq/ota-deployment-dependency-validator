# OTA Control Server (Spring Boot)

차량 OTA(Over-the-Air) 관제 서버 — Go/Python 프로토타입에서 Java/Spring Boot로 마이그레이션

## 마이그레이션 매핑

| 원본 (Go/Python) | → Spring Boot |
|---|---|
| `main.go` 엔트리포인트 | `OtaControlApplication.java` |
| `repository.NewHANARepository` | Spring Data JPA + `VehicleRepository` 등 |
| `transport.KafkaProducer` | `KafkaInventoryProducer` (@KafkaTemplate) |
| `transport.KafkaConsumer` | `KafkaInventoryConsumer` (@KafkaListener) |
| `transport.MQTTClient` | `MqttPublisher` (Paho Client) |
| `transport.StartCollector` | `MqttCollector` (MQTT→Kafka 브릿지) |
| `service.OTAAnalyzer` | `OtaAnalyzerService` |
| FastAPI `/check-update` | `OtaCheckController` |
| `validator.py` | `OtaValidatorService` + `Ecu.isCompatibleWith()` |
| `test_connection.py` | `DataInitializer` (local 프로필 자동 실행) |

## 로컬 실행

```bash
# 1. H2 인메모리 DB로 바로 실행 (Kafka/MQTT 없이)
./gradlew bootRun

# 2. 샘플 데이터 자동 삽입 확인
curl http://localhost:8080/api/vehicles
curl http://localhost:8080/api/vehicles/V001/ecus

# 3. OTA 의존성 검증 테스트
curl -X POST http://localhost:8080/api/ota/check-update \
  -H "Content-Type: application/json" \
  -d '{"vehicleId":"V001","packageId":"PKG_BMS_30"}'

# 4. H2 콘솔 접속
# http://localhost:8080/h2-console (JDBC URL: jdbc:h2:mem:otadb)
```

## Docker 실행 (전체 인프라)

```bash
cp .env.example .env
# .env 파일에 HANA DB 접속 정보 입력

docker compose up -d
```

## K8s 배포 (과제 3번)

```bash
# 1. 이미지 빌드
docker build -t ota-control-server:latest .

# 2. Secret 생성 (DB 접속 정보)
kubectl apply -f k8s/secret.yaml

# 3. 서버 배포
kubectl apply -f k8s/ota-server.yaml

# 4. nginx + NodePort 배포 (외부 접속)
kubectl apply -f k8s/nginx.yaml

# 5. 접속 확인
curl http://<EC2_PUBLIC_IP>:30080/api/ota
```

## API 목록

| Method | Path | 설명 |
|---|---|---|
| GET | `/api/ota` | 헬스체크 |
| POST | `/api/ota/check-update` | OTA 의존성 검증 |
| GET | `/api/vehicles` | 전체 차량 조회 |
| POST | `/api/vehicles` | 차량 등록 |
| GET | `/api/vehicles/{id}/ecus` | 차량 ECU 목록 |
| POST | `/api/firmware/upload-url` | 펌웨어 업로드 URL |
| GET | `/api/firmware/download-url` | 펌웨어 다운로드 URL |
| GET | `/api/campaigns` | 캠페인 목록 |
| POST | `/api/campaigns` | 캠페인 생성 |
| PUT | `/api/campaigns/{id}/start` | 캠페인 시작 |

## 과제 제출 매핑

- **과제 1번 (S3)**: `FirmwareController` + S3 정적 호스팅 (백오피스 프론트)
- **과제 2번 (DB)**: `VehicleController` + `OtaCheckController` → RDS/EC2 HANA DB
- **과제 3번 (K8s)**: `k8s/ota-server.yaml` + `k8s/nginx.yaml` → NodePort 30080
