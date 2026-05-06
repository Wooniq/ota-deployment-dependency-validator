package com.ota.campaign.messaging;

import com.ota.campaign.dto.CampaignEvent;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Service;

@Service
@RequiredArgsConstructor
@Slf4j
public class CampaignEventPublisher {

    private final KafkaTemplate<String, CampaignEvent> kafkaTemplate;

    @Value("${ota.kafka.topics.campaign-events:ota-campaign-events}")
    private String campaignEventsTopic;

    public void publishStarted(CampaignEvent event) {
        kafkaTemplate.send(campaignEventsTopic, String.valueOf(event.campaignId()), event)
                .whenComplete((result, ex) -> {
                    if (ex != null) {
                        log.error("[Kafka] Campaign event publish failed: campaignId={}, error={}",
                                event.campaignId(), ex.getMessage(), ex);
                        return;
                    }
                    log.info("[Kafka] Campaign event published: topic={}, campaignId={}",
                            campaignEventsTopic, event.campaignId());
                });
    }
}
