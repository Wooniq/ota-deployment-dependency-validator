package com.ota.control.service;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.ota.control.domain.Ecu;
import com.ota.control.domain.Vehicle;
import com.ota.control.messaging.MqttPublisher;
import com.ota.control.repository.EcuRepository;
import com.ota.control.repository.VehicleRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;
import java.time.Instant;

/**
 * OTA 분석 엔진
 * ← Go service.NewOTAAnalyzer(repo, serverMqttClient, adasVer, bmsVer) 포팅
 *
 * Kafka에서 수집된 차량 인벤토리 메시지를 분석하고,
 * 기준 버전 미달 시 MQTT를 통해 롤백 명령을 발송
 */
@Service
@RequiredArgsConstructor
@Slf4j
public class OtaAnalyzerService {

    private final EcuRepository ecuRepository;
    private final VehicleRepository vehicleRepository;
    private final MqttPublisher mqttPublisher;
    private final ObjectMapper objectMapper;

    @Value("${ota.baseline.adas-version:v2.2.2}")
    private String baselineAdasVersion;

    @Value("${ota.baseline.bms-version:v1.3.9}")
    private String baselineBmsVersion;

    /**
     * Kafka 메시지 분석 처리
     * Go의 kafkaConsumer.StartConsuming(ctx, analyzer) 콜백에 해당
     */
    @Transactional
    public void analyzeInventoryMessage(String message) {
        try {
            JsonNode node = objectMapper.readTree(message);
            String vehicleId = node.get("vehicle_id").asText();
            String ecuType = node.get("ecu_type").asText();
            int major = node.get("major").asInt();
            int minor = node.get("minor").asInt();
            int patch = node.get("patch").asInt();

            log.info("[분석] 수신: vehicle={}, ecu={}, version={}.{}.{}",
                    vehicleId, ecuType, major, minor, patch);

            // DB에 ECU 정보 upsert
            Ecu ecu = ecuRepository.findByVehicleIdAndEcuType(vehicleId, ecuType)
                    .orElse(Ecu.builder()
                            .vehicleId(vehicleId)
                            .ecuType(ecuType)
                            .build());

            ecu.setMajor(major);
            ecu.setMinor(minor);
            ecu.setPatch(patch);
            ecu.setLastReportedAt(Instant.now());
            ecuRepository.save(ecu);

            // 기준 버전 미달 시 롤백 커맨드 발송
            if (isVersionBelowBaseline(ecuType, major, minor, patch)) {
                log.warn("[분석] 기준 미달 감지! vehicle={}, ecu={} v{}.{}.{}",
                        vehicleId, ecuType, major, minor, patch);
                sendRollbackCommand(vehicleId, ecuType);
            }

        } catch (Exception e) {
            log.error("[분석] 메시지 처리 실패: {}", e.getMessage(), e);
        }
    }

    private boolean isVersionBelowBaseline(String ecuType, int major, int minor, int patch) {
        int[] baseline = parseVersion(switch (ecuType.toUpperCase()) {
            case "ADAS" -> baselineAdasVersion;
            case "BMS" -> baselineBmsVersion;
            default -> "v0.0.0";
        });

        // current < baseline
        if (major != baseline[0]) return major < baseline[0];
        if (minor != baseline[1]) return minor < baseline[1];
        return patch < baseline[2];
    }

    private int[] parseVersion(String version) {
        String v = version.startsWith("v") ? version.substring(1) : version;
        String[] parts = v.split("\\.");
        return new int[]{
                Integer.parseInt(parts[0]),
                Integer.parseInt(parts[1]),
                Integer.parseInt(parts[2])
        };
    }

    /**
     * MQTT를 통해 차량에 롤백 명령 발송
     * ← Go serverMqttClient를 통한 롤백 커맨드에 해당
     */
    private void sendRollbackCommand(String vehicleId, String ecuType) {
        try {
            String topic = "ota/command/" + vehicleId;
            String payload = objectMapper.writeValueAsString(new RollbackCommand(vehicleId, ecuType, "ROLLBACK"));
            mqttPublisher.publish(topic, payload);

            // 차량 상태 업데이트
            vehicleRepository.findById(vehicleId).ifPresent(v -> {
                v.setStatus(Vehicle.VehicleStatus.ROLLBACK);
                vehicleRepository.save(v);
            });

            log.info("[롤백] 명령 발송 완료: vehicle={}, ecu={}", vehicleId, ecuType);
        } catch (Exception e) {
            log.error("[롤백] 명령 발송 실패: {}", e.getMessage(), e);
        }
    }

    private record RollbackCommand(String vehicleId, String ecuType, String action) {}
}
