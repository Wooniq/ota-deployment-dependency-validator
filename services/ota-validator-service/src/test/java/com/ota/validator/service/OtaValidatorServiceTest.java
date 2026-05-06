package com.ota.validator.service;

import com.ota.validator.client.CampaignClient;
import com.ota.validator.client.InventoryClient;
import com.ota.validator.domain.ValidationHistory;
import com.ota.validator.dto.DependencyRuleResponse;
import com.ota.validator.dto.EcuInventoryResponse;
import com.ota.validator.dto.OtaCheckDto.CheckResponse;
import com.ota.validator.repository.ValidationHistoryRepository;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.test.util.ReflectionTestUtils;

import java.time.Instant;
import java.time.temporal.ChronoUnit;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class OtaValidatorServiceTest {

    @Test
    @DisplayName("외부 Campaign/Inventory API 응답으로 OTA 검증을 수행하고 이력 저장")
    void shouldValidateUsingExternalClients() {
        InventoryClient inventoryClient = mock(InventoryClient.class);
        CampaignClient campaignClient = mock(CampaignClient.class);
        ValidationHistoryRepository historyRepository = mock(ValidationHistoryRepository.class);
        OtaValidatorService service = new OtaValidatorService(inventoryClient, campaignClient, historyRepository);
        ReflectionTestUtils.setField(service, "maxInventoryAgeMinutes", 10L);

        when(inventoryClient.getEcusByVehicleId("V001")).thenReturn(List.of(
                new EcuInventoryResponse("V001", "BCM", 1, 5, 0, "1.5.0", Instant.now())
        ));
        when(campaignClient.getDependencyRules("PKG_BMS_30")).thenReturn(List.of(
                new DependencyRuleResponse(1L, "PKG_BMS_30", "BCM", 1, 2, 0, "1.2.0")
        ));

        CheckResponse response = service.validateUpdate("V001", "PKG_BMS_30");

        assertThat(response.isAvailable()).isTrue();
        assertThat(response.getDetails().get(0).getStatus()).isEqualTo("PASS");

        ArgumentCaptor<ValidationHistory> captor = ArgumentCaptor.forClass(ValidationHistory.class);
        verify(historyRepository).save(captor.capture());
        assertThat(captor.getValue().getStatus()).isEqualTo(ValidationHistory.ValidationStatus.PASS);
    }

    @Test
    @DisplayName("오래된 인벤토리는 STALE_INVENTORY로 차단")
    void shouldRejectStaleInventory() {
        InventoryClient inventoryClient = mock(InventoryClient.class);
        CampaignClient campaignClient = mock(CampaignClient.class);
        ValidationHistoryRepository historyRepository = mock(ValidationHistoryRepository.class);
        OtaValidatorService service = new OtaValidatorService(inventoryClient, campaignClient, historyRepository);
        ReflectionTestUtils.setField(service, "maxInventoryAgeMinutes", 10L);

        when(inventoryClient.getEcusByVehicleId("V001")).thenReturn(List.of(
                new EcuInventoryResponse("V001", "BCM", 1, 5, 0, "1.5.0",
                        Instant.now().minus(30, ChronoUnit.MINUTES))
        ));
        when(campaignClient.getDependencyRules("PKG_BMS_30")).thenReturn(List.of(
                new DependencyRuleResponse(1L, "PKG_BMS_30", "BCM", 1, 2, 0, "1.2.0")
        ));

        CheckResponse response = service.validateUpdate("V001", "PKG_BMS_30");

        assertThat(response.isAvailable()).isFalse();
        assertThat(response.getDetails().get(0).getReasonCode()).isEqualTo("STALE_INVENTORY");
    }
}
