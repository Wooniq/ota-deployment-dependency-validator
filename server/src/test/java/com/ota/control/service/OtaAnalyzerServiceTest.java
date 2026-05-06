package com.ota.control.service;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.ota.control.domain.Ecu;
import com.ota.control.messaging.MqttPublisher;
import com.ota.control.repository.EcuRepository;
import com.ota.control.repository.VehicleRepository;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.mockito.ArgumentCaptor;

import java.time.Instant;
import java.util.Optional;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.*;

class OtaAnalyzerServiceTest {

    @Test
    @DisplayName("동일 차량/ECU 인벤토리 재수신 시 기존 행을 갱신")
    void shouldUpdateExistingEcuInventoryIdempotently() {
        // given
        EcuRepository ecuRepository = mock(EcuRepository.class);
        VehicleRepository vehicleRepository = mock(VehicleRepository.class);
        MqttPublisher mqttPublisher = mock(MqttPublisher.class);
        OtaAnalyzerService service = new OtaAnalyzerService(
                ecuRepository,
                vehicleRepository,
                mqttPublisher,
                new ObjectMapper()
        );

        Ecu existing = Ecu.builder()
                .id(10L)
                .vehicleId("V001")
                .ecuType("BCM")
                .major(1).minor(0).patch(0)
                .lastReportedAt(Instant.now().minusSeconds(60))
                .build();

        when(ecuRepository.findByVehicleIdAndEcuType("V001", "BCM")).thenReturn(Optional.of(existing));

        String message = """
                {"vehicle_id":"V001","ecu_type":"BCM","major":1,"minor":5,"patch":0}
                """;

        // when
        service.analyzeInventoryMessage(message);

        // then
        ArgumentCaptor<Ecu> captor = ArgumentCaptor.forClass(Ecu.class);
        verify(ecuRepository).save(captor.capture());

        Ecu saved = captor.getValue();
        assertThat(saved.getId()).isEqualTo(10L);
        assertThat(saved.getMajor()).isEqualTo(1);
        assertThat(saved.getMinor()).isEqualTo(5);
        assertThat(saved.getPatch()).isEqualTo(0);
        assertThat(saved.getLastReportedAt()).isNotNull();
        verifyNoInteractions(mqttPublisher);
    }
}
