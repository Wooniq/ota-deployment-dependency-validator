# 1단계: 빌드 스테이지
FROM gradle:8.10-jdk17 AS builder

WORKDIR /app

# 의존성 캐시 레이어
COPY build.gradle settings.gradle ./
RUN gradle dependencies --no-daemon || true

# 소스 복사 및 빌드
COPY src ./src
RUN gradle bootJar --no-daemon

# ---------------------------------------------------------

# 2단계: 실행 스테이지
FROM eclipse-temurin:17-jre-alpine

RUN apk --no-cache add curl tzdata

WORKDIR /app

# 빌드된 JAR 복사
COPY --from=builder /app/build/libs/*.jar app.jar

# 헬스체크
HEALTHCHECK --interval=30s --timeout=3s \
    CMD curl -f http://localhost:8080/api/ota || exit 1

EXPOSE 8080

ENTRYPOINT ["java", "-jar", "app.jar"]
