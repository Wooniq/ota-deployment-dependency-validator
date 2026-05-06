package com.ota.campaign.controller;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.ota.campaign.domain.Campaign;
import com.ota.campaign.domain.Campaign.CampaignStatus;
import com.ota.campaign.domain.CampaignTarget;
import com.ota.campaign.domain.UpdatePackage;
import com.ota.campaign.dto.CampaignEvent;
import com.ota.campaign.messaging.CampaignEventPublisher;
import com.ota.campaign.repository.CampaignRepository;
import com.ota.campaign.repository.CampaignTargetRepository;
import com.ota.campaign.repository.UpdatePackageRepository;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import java.util.List;
import java.util.Map;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.put;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest({CampaignController.class, HealthController.class})
class CampaignControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @Autowired
    private ObjectMapper objectMapper;

    @MockBean
    private CampaignRepository campaignRepository;

    @MockBean
    private UpdatePackageRepository packageRepository;

    @MockBean
    private CampaignTargetRepository targetRepository;

    @MockBean
    private CampaignEventPublisher campaignEventPublisher;

    @Test
    @DisplayName("캠페인 시작 시 ota-campaign-events payload를 발행")
    void shouldPublishCampaignEventWhenCampaignStarts() throws Exception {
        UpdatePackage updatePackage = UpdatePackage.builder()
                .packageId("PKG_BMS_30")
                .targetEcuType("BMS")
                .major(3).minor(0).patch(0)
                .build();
        Campaign campaign = Campaign.builder()
                .id(7L)
                .campaignName("BMS rollout")
                .updatePackage(updatePackage)
                .status(CampaignStatus.CREATED)
                .totalVehicles(2)
                .build();

        when(campaignRepository.findById(7L)).thenReturn(Optional.of(campaign));
        when(campaignRepository.save(any(Campaign.class))).thenAnswer(invocation -> invocation.getArgument(0));
        when(targetRepository.findByCampaignId(7L)).thenReturn(List.of(
                CampaignTarget.builder().id(1L).campaign(campaign).vehicleId("V001").status(CampaignTarget.TargetStatus.PENDING).build(),
                CampaignTarget.builder().id(2L).campaign(campaign).vehicleId("V002").status(CampaignTarget.TargetStatus.PENDING).build()
        ));

        mockMvc.perform(put("/api/campaigns/7/start")
                        .contentType(MediaType.APPLICATION_JSON)
                        .content(objectMapper.writeValueAsString(Map.of(
                                "targetVehicleIds", List.of("V001", "V002")
                        ))))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("IN_PROGRESS"));

        ArgumentCaptor<CampaignEvent> captor = ArgumentCaptor.forClass(CampaignEvent.class);
        verify(campaignEventPublisher).publishStarted(captor.capture());
        assertThat(captor.getValue().campaignId()).isEqualTo(7L);
        assertThat(captor.getValue().packageId()).isEqualTo("PKG_BMS_30");
        assertThat(captor.getValue().targetVehicleIds()).containsExactly("V001", "V002");
    }
}
