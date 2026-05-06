# OTA Inventory Service

차량과 ECU 인벤토리를 소유하는 독립 Spring Boot 서비스입니다.

## API

| Method | Path | Description |
|---|---|---|
| GET | `/api/vehicles/health` | Health check |
| GET | `/api/vehicles` | Vehicle list |
| GET | `/api/vehicles/{vehicleId}/ecus` | Vehicle ECU list |
| POST | `/api/vehicles/{vehicleId}/ecus` | Idempotent ECU upsert |
| GET | `/internal/vehicles/{vehicleId}/ecus` | ECU inventory for validator service |
