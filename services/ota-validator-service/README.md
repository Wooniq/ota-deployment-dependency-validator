# OTA Validator Service

Campaign Service와 Inventory Service의 내부 API를 호출해 OTA 사전 검증을 수행하고 검증 이력을 저장하는 독립 Spring Boot 서비스입니다.

## Run

```bash
sh ../../server/gradlew bootRun
```

## API

| Method | Path | Description |
|---|---|---|
| GET | `/api/ota/health` | Health check |
| POST | `/api/ota/check-update` | OTA dependency validation |
| GET | `/api/validation-histories` | Validation history list |
| GET | `/api/validation-histories/vehicles/{vehicleId}` | Vehicle validation history |
