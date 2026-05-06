# OTA Firmware Service

S3 Presigned URL 기반 펌웨어 업로드, 다운로드, 삭제 API를 제공하는 독립 Spring Boot 서비스입니다.

## Run

```bash
sh ../../server/gradlew bootRun
```

## API

| Method | Path | Description |
|---|---|---|
| GET | `/api/firmware/health` | Health check |
| POST | `/api/firmware/upload-url?key=firmware/BMS/v3.0.0.bin` | Upload presigned URL |
| GET | `/api/firmware/download-url?key=firmware/BMS/v3.0.0.bin` | Download presigned URL |
| DELETE | `/api/firmware?key=firmware/BMS/v3.0.0.bin` | Delete firmware object |
