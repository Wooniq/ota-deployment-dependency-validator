package com.ota.inventory.controller;

import com.ota.inventory.domain.Ecu;
import com.ota.inventory.repository.EcuRepository;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import java.time.Instant;
import java.util.List;

import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest(InternalInventoryController.class)
class InternalInventoryControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private EcuRepository ecuRepository;

    @Test
    @DisplayName("validator-service용 ECU 인벤토리를 반환")
    void shouldReturnEcusForValidator() throws Exception {
        when(ecuRepository.findByVehicleId("V001")).thenReturn(List.of(
                Ecu.builder()
                        .vehicleId("V001")
                        .ecuType("BCM")
                        .major(1).minor(5).patch(0)
                        .lastReportedAt(Instant.parse("2026-05-06T00:00:00Z"))
                        .build()
        ));

        mockMvc.perform(get("/internal/vehicles/V001/ecus"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$[0].vehicleId").value("V001"))
                .andExpect(jsonPath("$[0].ecuType").value("BCM"))
                .andExpect(jsonPath("$[0].version").value("1.5.0"))
                .andExpect(jsonPath("$[0].lastReportedAt").value("2026-05-06T00:00:00Z"));
    }
}
