package com.ota.control.service;

import com.ota.control.domain.DependencyRule;
import com.ota.control.domain.Ecu;
import com.ota.control.dto.OtaCheckDto.*;
import com.ota.control.repository.DependencyRuleRepository;
import com.ota.control.repository.EcuRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.util.ArrayList;
import java.util.List;

/**
 * OTA 의존성 검증 엔진
 * ← Python validator.py의 validate_ota() + is_compatible() 포팅
 *
 * 차량의 현재 ECU SW 버전과 업데이트 패키지의 의존성 규칙을 비교하여
 * 배포 가능 여부를 판별
 */
@Service
@RequiredArgsConstructor
@Slf4j
public class OtaValidatorService {

    private final EcuRepository ecuRepository;
    private final DependencyRuleRepository ruleRepository;

    /**
     * 차량에 대한 패키지 업데이트 가능 여부 검증
     */
    @Transactional(readOnly = true)
    public CheckResponse validateUpdate(String vehicleId, String packageId) {
        List<Ecu> vehicleEcus = ecuRepository.findByVehicleId(vehicleId);
        List<DependencyRule> rules = ruleRepository.findByUpdatePackagePackageId(packageId);

        List<RuleResult> report = new ArrayList<>();

        for (DependencyRule rule : rules) {
            // 1. 대상 ECU 탐색 (Python: next((e for e in vehicle_ecus if ...), None))
            Ecu targetEcu = vehicleEcus.stream()
                    .filter(e -> e.getEcuType().equals(rule.getRequiredType()))
                    .findFirst()
                    .orElse(null);

            // 2. ECU가 누락된 경우
            if (targetEcu == null) {
                report.add(RuleResult.builder()
                        .rule(rule.getRequiredType())
                        .status("FAIL")
                        .reason("검증에 필요한 제어기를 차량에서 찾을 수 없습니다.")
                        .build());
                continue;
            }

            // 3. 버전 비교 (Python: is_compatible(current, required))
            boolean compatible = targetEcu.isCompatibleWith(
                    rule.getMinMajor(), rule.getMinMinor(), rule.getMinPatch()
            );

            if (compatible) {
                report.add(RuleResult.builder()
                        .rule(rule.getRequiredType())
                        .status("PASS")
                        .currentVersion(targetEcu.getVersionString())
                        .requiredVersion(rule.getRequiredVersionString())
                        .build());
            } else {
                report.add(RuleResult.builder()
                        .rule(rule.getRequiredType())
                        .status("FAIL")
                        .currentVersion(targetEcu.getVersionString())
                        .requiredVersion(rule.getRequiredVersionString())
                        .reason(String.format(
                                "버전 미달 (현재: v%s < 요구: v%s)",
                                targetEcu.getVersionString(),
                                rule.getRequiredVersionString()
                        ))
                        .build());
            }
        }

        // 4. 최종 판결 (Python: all(item['status'] == "PASS" for item in report))
        boolean allPassed = report.stream().allMatch(r -> "PASS".equals(r.getStatus()));

        log.info("[OTA 검증] vehicle={}, package={}, result={}", vehicleId, packageId, allPassed);

        return CheckResponse.builder()
                .vehicleId(vehicleId)
                .packageId(packageId)
                .available(allPassed)
                .details(report)
                .build();
    }
}
