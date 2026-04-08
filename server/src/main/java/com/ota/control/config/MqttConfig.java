package com.ota.control.config;

import lombok.extern.slf4j.Slf4j;
import org.eclipse.paho.client.mqttv3.MqttClient;
import org.eclipse.paho.client.mqttv3.MqttConnectOptions;
import org.eclipse.paho.client.mqttv3.persist.MemoryPersistence;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
@Slf4j
public class MqttConfig {

    @Value("${mqtt.broker-url:tcp://localhost:1883}")
    private String brokerUrl;

    @Value("${mqtt.client-id:OTA-Server-Commander}")
    private String clientId;

    @Bean
    public MqttClient mqttClient() throws Exception {
        MqttConnectOptions options = new MqttConnectOptions();
        options.setAutomaticReconnect(true);
        options.setCleanSession(true);
        options.setConnectionTimeout(10);

        MqttClient client = new MqttClient(brokerUrl, clientId, new MemoryPersistence());

        try {
            client.connect(options);
            log.info("[MQTT] 브로커 연결 성공: {}", brokerUrl);
        } catch (Exception e) {
            log.warn("[MQTT] 브로커 연결 실패 (서버는 계속 기동): {}", e.getMessage());
        }

        return client;
    }
}
