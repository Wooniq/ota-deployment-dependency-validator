# 1단계: 빌드 스테이지 (Build Stage)
FROM golang:1.22-alpine AS builder

# 작업 디렉토리 설정
WORKDIR /app

# 1. 의존성 파일을 먼저 복사하여 레이어 캐싱 활용 (DevOps 최적화)
COPY go.mod ./
# 만약 go.sum 파일이 있다면 아래 주석을 해제하세요.
# COPY go.sum ./
RUN go mod download

# 2. 소스 코드 복사 및 정적 빌드
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o ota-agent main.go

# ---------------------------------------------------------

# 2단계: 실행 스테이지 (Final Stage)
FROM alpine:latest

# 실행에 필요한 최소 패키지만 설치
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# 1단계에서 빌드된 바이너리만 복사 (최종 이미지 용량 약 5MB)
COPY --from=builder /app/ota-agent .

# 실행 권한 부여
RUN chmod +x ota-agent

# 에이전트 실행
ENTRYPOINT ["./ota-agent"]