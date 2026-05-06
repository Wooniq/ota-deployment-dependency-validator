# Services

Strangler Fig 방식으로 기존 `server/` 모놀리스를 점진 분리하는 마이크로서비스 작업 공간입니다.

현재 `server/`는 그대로 유지하며, 독립성이 높은 기능부터 `services/` 아래로 추출합니다.

| Service | Port | Status |
|---|---:|---|
| `ota-firmware-service` | 8084 | S3 Presigned URL API 추출 완료 |
| `ota-campaign-service` | 8083 | Campaign API, internal package API, `ota-campaign-events` 발행 구조 추출 완료 |
| `ota-validator-service` | 8082 | 외부 Campaign/Inventory API 기반 OTA 검증, ValidationHistory 저장 구조 추출 완료 |
| `ota-inventory-service` | 8081 | Vehicle/ECU 소유, idempotent ECU upsert, internal ECU API 추출 완료 |
| `ota-notification-svc` | 8085 | 예정 |
