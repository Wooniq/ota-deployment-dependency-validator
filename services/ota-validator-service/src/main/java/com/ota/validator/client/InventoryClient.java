package com.ota.validator.client;

import com.ota.validator.dto.EcuInventoryResponse;
import lombok.RequiredArgsConstructor;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.core.ParameterizedTypeReference;
import org.springframework.stereotype.Component;
import org.springframework.web.client.RestClient;

import java.util.List;

@Component
@RequiredArgsConstructor
public class InventoryClient {

    private final RestClient.Builder restClientBuilder;

    @Value("${service.inventory.url}")
    private String inventoryServiceUrl;

    public List<EcuInventoryResponse> getEcusByVehicleId(String vehicleId) {
        return restClientBuilder.baseUrl(inventoryServiceUrl).build()
                .get()
                .uri("/internal/vehicles/{vehicleId}/ecus", vehicleId)
                .retrieve()
                .body(new ParameterizedTypeReference<>() {});
    }
}
