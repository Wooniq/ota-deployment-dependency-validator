package com.ota.validator.client;

import com.ota.validator.dto.DependencyRuleResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;

import java.util.List;

@Component
@RequiredArgsConstructor
public class CampaignClient {

    private final RestClient.Builder restClientBuilder;

    @Value("${service.campaign.url}")
    private String campaignServiceUrl;

    public List<DependencyRuleResponse> getDependencyRules(String packageId) {
        return restClientBuilder.baseUrl(campaignServiceUrl).build()
                .get()
                .uri("/internal/packages/{packageId}/rules", packageId)
                .retrieve()
                .body(new ParameterizedTypeReference<>() {});
    }
}
