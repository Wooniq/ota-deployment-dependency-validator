package com.ota.control.messaging;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Component;

/**
 * Kafka Producer (수집용)
 * ← Go transport.NewKafkaProducer(brokers, "ota-inventory") 포팅
 */
@Component
@RequiredArgsConstructor
@Slf4j
public class KafkaInventoryProducer {

    private final KafkaTemplate<String, String> kafkaTemplate;

    private static final String TOPIC = "ota-inventory";

    /**
     * 차량 인벤토리 데이터를 Kafka로 전송
     */
    public void send(String vehicleId, String payload) {
        kafkaTemplate.send(TOPIC, vehicleId, payload)
                .whenComplete((result, ex) -> {
                    if (ex != null) {
                        log.error("[Kafka] 전송 실패: vehicle={}, error={}", vehicleId, ex.getMessage());
                    } else {
                        log.debug("[Kafka] 전송 성공: vehicle={}, offset={}",
                                vehicleId, result.getRecordMetadata().offset());
                    }
                });
    }
}
