package com.ota.control.messaging;

import com.ota.control.service.OtaAnalyzerService;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.kafka.annotation.KafkaListener;
import org.springframework.stereotype.Component;

/**
 * Kafka Consumer (분석 계층)
 * ← Go transport.NewKafkaConsumer(brokers, "ota-inventory", "analyzer-group-v1")
 *    + kafkaConsumer.StartConsuming(ctx, analyzer) 포팅
 */
@Component
@RequiredArgsConstructor
@Slf4j
public class KafkaInventoryConsumer {

    private final OtaAnalyzerService analyzerService;

    @KafkaListener(
            topics = "ota-inventory",
            groupId = "analyzer-group-v1",
            autoStartup = "${spring.kafka.listener.auto-startup:true}"  // ← yml 값 참조
    )
    public void consume(String message) {
        log.debug("[Kafka] 메시지 수신: {}", message);
        analyzerService.analyzeInventoryMessage(message);
    }
}
