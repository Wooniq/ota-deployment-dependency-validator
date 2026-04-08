package com.ota.control.messaging;

import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.eclipse.paho.client.mqttv3.*;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.context.event.ApplicationReadyEvent;
import org.springframework.context.event.EventListener;
import org.springframework.stereotype.Component;

/**
 * MQTT 수집기 (입력 → Kafka)
 * ← Go transport.StartCollector(mqttBroker, kafkaProducer) 포팅
 *
 * 차량 에이전트가 MQTT로 발행한 인벤토리 데이터를 구독하여
 * Kafka 토픽으로 전달하는 브릿지 역할
 */
@Component
@RequiredArgsConstructor
@Slf4j
public class MqttCollector {

    private final MqttClient mqttClient;
    private final KafkaInventoryProducer kafkaProducer;

    @Value("${mqtt.topics.inventory:ota/inventory}")
    private String inventoryTopic;

    @EventListener(ApplicationReadyEvent.class)
    public void startCollecting() {
        try {
            mqttClient.subscribe(inventoryTopic + "/#", 1, (topic, message) -> {
                String payload = new String(message.getPayload());
                log.debug("[Collector] MQTT 수신: topic={}, payload={}", topic, payload);

                // 토픽에서 vehicleId 추출: ota/inventory/{vehicleId}
                String[] parts = topic.split("/");
                String vehicleId = parts.length >= 3 ? parts[2] : "unknown";

                // Kafka로 전달
                kafkaProducer.send(vehicleId, payload);
            });

            log.info("[Collector] MQTT 수집기 가동: topic={}", inventoryTopic + "/#");
        } catch (MqttException e) {
            log.error("[Collector] MQTT 구독 실패: {}", e.getMessage(), e);
        }
    }
}
