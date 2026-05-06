package com.ota.campaign.controller;

import com.ota.campaign.domain.DependencyRule;
import com.ota.campaign.domain.UpdatePackage;
import com.ota.campaign.repository.DependencyRuleRepository;
import com.ota.campaign.repository.UpdatePackageRepository;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import java.util.List;
import java.util.Optional;

import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(InternalPackageController.class)
class InternalPackageControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private UpdatePackageRepository packageRepository;

    @MockBean
    private DependencyRuleRepository ruleRepository;

    @Test
    @DisplayName("validator-service용 패키지 의존성 규칙을 DTO로 반환")
    void shouldReturnDependencyRulesForValidator() throws Exception {
        UpdatePackage updatePackage = UpdatePackage.builder()
                .packageId("PKG_BMS_30")
                .targetEcuType("BMS")
                .major(3).minor(0).patch(0)
                .build();
        DependencyRule rule = DependencyRule.builder()
                .ruleId(1L)
                .updatePackage(updatePackage)
                .requiredType("BCM")
                .minMajor(1).minMinor(2).minPatch(0)
                .build();

        when(ruleRepository.findByUpdatePackagePackageId("PKG_BMS_30")).thenReturn(List.of(rule));

        mockMvc.perform(get("/internal/packages/PKG_BMS_30/rules"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].packageId").value("PKG_BMS_30"))
                .andExpect(jsonPath("$[0].requiredType").value("BCM"))
                .andExpect(jsonPath("$[0].requiredVersion").value("1.2.0"));
    }

    @Test
    @DisplayName("validator-service용 패키지 정보를 DTO로 반환")
    void shouldReturnPackageForValidator() throws Exception {
        UpdatePackage updatePackage = UpdatePackage.builder()
                .packageId("PKG_BMS_30")
                .targetEcuType("BMS")
                .major(3).minor(0).patch(0)
                .s3Key("firmware/BMS/v3.0.0.bin")
                .build();

        when(packageRepository.findById("PKG_BMS_30")).thenReturn(Optional.of(updatePackage));

        mockMvc.perform(get("/internal/packages/PKG_BMS_30"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.packageId").value("PKG_BMS_30"))
                .andExpect(jsonPath("$.version").value("3.0.0"))
                .andExpect(jsonPath("$.s3Key").value("firmware/BMS/v3.0.0.bin"));
    }
}
