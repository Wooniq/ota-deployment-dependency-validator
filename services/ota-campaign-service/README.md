# OTA Campaign Service

캠페인, 업데이트 패키지, 의존성 규칙을 소유하는 독립 Spring Boot 서비스입니다.

캠페인 시작 시 `ota-campaign-events` Kafka 토픽으로 시작 이벤트를 발행합니다.

## Run

```bash
sh ../../server/gradlew bootRun
```

## API

| Method | Path | Description |
|---|---|---|
| GET | `/api/campaigns/health` | Health check |
| GET | `/api/campaigns` | Campaign list |
| POST | `/api/campaigns` | Create campaign |
| PUT | `/api/campaigns/{id}/start` | Start campaign and publish `ota-campaign-events` |
| PUT | `/api/campaigns/{id}/abort` | Abort campaign |
| GET | `/internal/packages/{packageId}` | Package lookup for validator service |
| GET | `/internal/packages/{packageId}/rules` | Dependency rules for validator service |
