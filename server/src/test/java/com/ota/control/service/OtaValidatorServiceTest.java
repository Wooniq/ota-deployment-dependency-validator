package com.ota.control.service;

import com.ota.control.domain.DependencyRule;
import com.ota.control.domain.Ecu;
import com.ota.control.domain.UpdatePackage;
import com.ota.control.dto.OtaCheckDto.CheckResponse;
import com.ota.control.repository.DependencyRuleRepository;
import com.ota.control.repository.EcuRepository;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.when;

/**
 * Python validator.py의 테스트 시나리오를 Java로 재현
 * - V001 (BCM 1.5.0) → BCM ≥ 1.2.0 규칙 PASS
 * - V002 (BCM 1.0.0) → BCM ≥ 1.2.0 규칙 FAIL
 */
@ExtendWith(MockitoExtension.class)
class OtaValidatorServiceTest {

    @Mock
    private EcuRepository ecuRepository;

    @Mock
    private DependencyRuleRepository ruleRepository;

    @InjectMocks
    private OtaValidatorService validatorService;

    @Test
    @DisplayName("V001 (BCM 1.5.0) - 의존성 충족 → 업데이트 가능")
    void shouldPassWhenVersionMeetsRequirement() {
        // given: V001의 ECU 현황 (test_connection.py 데이터)
        List<Ecu> ecus = List.of(
                Ecu.builder().vehicleId("V001").ecuType("BMS").major(2).minor(0).patch(0).build(),
                Ecu.builder().vehicleId("V001").ecuType("BCM").major(1).minor(5).patch(0).build()
        );

        UpdatePackage pkg = UpdatePackage.builder().packageId("PKG_BMS_30").build();
        List<DependencyRule> rules = List.of(
                DependencyRule.builder()
                        .updatePackage(pkg)
                        .requiredType("BCM")
                        .minMajor(1).minMinor(2).minPatch(0)
                        .build()
        );

        when(ecuRepository.findByVehicleId("V001")).thenReturn(ecus);
        when(ruleRepository.findByUpdatePackagePackageId("PKG_BMS_30")).thenReturn(rules);

        // when
        CheckResponse result = validatorService.validateUpdate("V001", "PKG_BMS_30");

        // then
        assertThat(result.isAvailable()).isTrue();
        assertThat(result.getDetails()).hasSize(1);
        assertThat(result.getDetails().get(0).getStatus()).isEqualTo("PASS");
        assertThat(result.getDetails().get(0).getCurrentVersion()).isEqualTo("1.5.0");
    }

    @Test
    @DisplayName("V002 (BCM 1.0.0) - 의존성 미달 → 업데이트 불가")
    void shouldFailWhenVersionBelowRequirement() {
        // given: V002의 ECU 현황 (BCM 1.0.0 < 요구 1.2.0)
        List<Ecu> ecus = List.of(
                Ecu.builder().vehicleId("V002").ecuType("BMS").major(2).minor(0).patch(0).build(),
                Ecu.builder().vehicleId("V002").ecuType("BCM").major(1).minor(0).patch(0).build()
        );

        UpdatePackage pkg = UpdatePackage.builder().packageId("PKG_BMS_30").build();
        List<DependencyRule> rules = List.of(
                DependencyRule.builder()
                        .updatePackage(pkg)
                        .requiredType("BCM")
                        .minMajor(1).minMinor(2).minPatch(0)
                        .build()
        );

        when(ecuRepository.findByVehicleId("V002")).thenReturn(ecus);
        when(ruleRepository.findByUpdatePackagePackageId("PKG_BMS_30")).thenReturn(rules);

        // when
        CheckResponse result = validatorService.validateUpdate("V002", "PKG_BMS_30");

        // then
        assertThat(result.isAvailable()).isFalse();
        assertThat(result.getDetails()).hasSize(1);
        assertThat(result.getDetails().get(0).getStatus()).isEqualTo("FAIL");
        assertThat(result.getDetails().get(0).getReason()).contains("버전 미달");
    }

    @Test
    @DisplayName("ECU 누락 시 FAIL 처리")
    void shouldFailWhenEcuNotFound() {
        // given: ECU가 없는 차량
        List<Ecu> ecus = List.of(
                Ecu.builder().vehicleId("V003").ecuType("BMS").major(2).minor(0).patch(0).build()
        );

        UpdatePackage pkg = UpdatePackage.builder().packageId("PKG_BMS_30").build();
        List<DependencyRule> rules = List.of(
                DependencyRule.builder()
                        .updatePackage(pkg)
                        .requiredType("BCM")  // BCM이 없음
                        .minMajor(1).minMinor(2).minPatch(0)
                        .build()
        );

        when(ecuRepository.findByVehicleId("V003")).thenReturn(ecus);
        when(ruleRepository.findByUpdatePackagePackageId("PKG_BMS_30")).thenReturn(rules);

        // when
        CheckResponse result = validatorService.validateUpdate("V003", "PKG_BMS_30");

        // then
        assertThat(result.isAvailable()).isFalse();
        assertThat(result.getDetails().get(0).getReason()).contains("찾을 수 없습니다");
    }
}
