# 1단계: 빌드 스테이지
FROM golang:1.25-alpine AS builder

WORKDIR /app

# 의존성 로드
COPY cmd/server/go.mod cmd/server/go.sum ./

# 의존성 다운로드
RUN go mod download

# 소스 전체 복사
COPY . .

# 서버 바이너리 빌드
RUN CGO_ENABLED=0 GOOS=linux go build -o ota-server ./cmd/server

# ---------------------------------------------------------

# 2단계: 실행 스테이지
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 빌드된 서버 실행 파일 복사
COPY --from=builder /app/ota-server .

# .env 파일을 찾기 위해 필요한 경로 설정 (없어도 실행은 되게끔)
# 실제 환경 변수는 docker-compose의 env_file을 통해 주입됩니다.

ENTRYPOINT ["./ota-server"]
