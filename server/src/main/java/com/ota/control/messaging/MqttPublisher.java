package com.ota.control.messaging;

import lombok.extern.slf4j.Slf4j;
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttMessage;
import org.springframework.stereotype.Component;

/**
 * MQTT 명령 발송기
 * ← Go transport.NewMQTTClient(mqttBroker, "OTA-Server-Commander") 포팅
 *
 * 관제 서버 → 차량 에이전트 방향 커맨드 (롤백 등) 발송용
 */
@Component
@Slf4j
public class MqttPublisher {

    private final MqttClient mqttClient;

    public MqttPublisher(MqttClient mqttClient) {
        this.mqttClient = mqttClient;
    }

    public void publish(String topic, String payload) {
        try {
            MqttMessage message = new MqttMessage(payload.getBytes());
            message.setQos(1);
            mqttClient.publish(topic, message);
            log.info("[MQTT] 발행 완료: topic={}", topic);
        } catch (Exception e) {
            log.error("[MQTT] 발행 실패: topic={}, error={}", topic, e.getMessage(), e);
        }
    }
}
