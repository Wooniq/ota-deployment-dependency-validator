FROM gradle:8.10-jdk17 AS builder

WORKDIR /app

COPY build.gradle settings.gradle ./
RUN gradle dependencies --no-daemon || true

COPY src ./src
RUN gradle bootJar --no-daemon

FROM eclipse-temurin:17-jre-alpine

RUN apk --no-cache add curl tzdata

WORKDIR /app

COPY --from=builder /app/build/libs/*.jar app.jar

HEALTHCHECK --interval=30s --timeout=3s \
    CMD curl -f http://localhost:8082/api/ota/health || exit 1

EXPOSE 8082

ENTRYPOINT ["java", "-jar", "app.jar"]
