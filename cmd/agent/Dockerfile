# 1단계: 빌드 스테이지 (Go 1.25 환경)

FROM golang:1.25-alpine AS builder

# 작업 디렉토리 설정
WORKDIR /app

# 1. 의존성 파일을 먼저 복사 (프로젝트 루트에서 실행하므로 경로 명시)
COPY cmd/agent/go.mod ./
# COPY cmd/agent/go.sum ./  # 파일이 있다면 주석 해제
RUN go mod download

# 2. 프로젝트 전체 소스 코드 복사 (pkg 등 상위 폴더 포함)
COPY . .

# 3. 바이너리 빌드 (디렉토리 경로 ./cmd/agent 지정으로 인자 오류 해결)
RUN CGO_ENABLED=0 GOOS=linux go build -o ota-agent ./cmd/agent

# ---------------------------------------------------------

# 2단계: 실행 스테이지 (초경량 Alpine)
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 빌드된 실행 파일만 복사
COPY --from=builder /app/ota-agent .
RUN chmod +x ota-agent

# 실행 시 환경 변수로부터 MQTT 주소 등을 받을 수 있게 설정
ENTRYPOINT ["./ota-agent"]