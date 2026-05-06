package com.ota.validator.service;

import com.ota.validator.client.CampaignClient;
import com.ota.validator.client.InventoryClient;
import com.ota.validator.domain.ValidationHistory;
import com.ota.validator.domain.ValidationHistory.FailureReason;
import com.ota.validator.domain.ValidationHistory.ValidationStatus;
import com.ota.validator.dto.DependencyRuleResponse;
import com.ota.validator.dto.EcuInventoryResponse;
import com.ota.validator.dto.OtaCheckDto.CheckResponse;
import com.ota.validator.dto.OtaCheckDto.RuleResult;
import com.ota.validator.repository.ValidationHistoryRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import org.springframework.transaction.annotation.Transactional;

import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;

@Service
@RequiredArgsConstructor
@Slf4j
public class OtaValidatorService {

    private final InventoryClient inventoryClient;
    private final CampaignClient campaignClient;
    private final ValidationHistoryRepository validationHistoryRepository;

    @Value("${ota.inventory.max-age-minutes:10}")
    private long maxInventoryAgeMinutes = 10;

    @Transactional
    public CheckResponse validateUpdate(String vehicleId, String packageId) {
        List<EcuInventoryResponse> vehicleEcus = inventoryClient.getEcusByVehicleId(vehicleId);
        List<DependencyRuleResponse> rules = campaignClient.getDependencyRules(packageId);
        List<RuleResult> report = new ArrayList<>();

        for (DependencyRuleResponse rule : rules) {
            EcuInventoryResponse targetEcu = vehicleEcus.stream()
                    .filter(ecu -> ecu.ecuType().equals(rule.requiredType()))
                    .findFirst()
                    .orElse(null);

            if (targetEcu == null) {
                RuleResult result = fail(rule.requiredType(), null, null,
                        FailureReason.MISSING_REQUIRED_ECU,
                        "검증에 필요한 제어기를 차량에서 찾을 수 없습니다.");
                report.add(result);
                recordHistory(vehicleId, packageId, rule.requiredType(), result, FailureReason.MISSING_REQUIRED_ECU);
                continue;
            }

            if (isStale(targetEcu)) {
                RuleResult result = fail(rule.requiredType(), targetEcu.versionString(), rule.requiredVersionString(),
                        FailureReason.STALE_INVENTORY,
                        "최신 인벤토리가 아닙니다. 마지막 보고 시각: " + targetEcu.lastReportedAt());
                report.add(result);
                recordHistory(vehicleId, packageId, rule.requiredType(), result, FailureReason.STALE_INVENTORY);
                continue;
            }

            boolean compatible = targetEcu.isCompatibleWith(rule.minMajor(), rule.minMinor(), rule.minPatch());
            if (compatible) {
                RuleResult result = RuleResult.builder()
                        .rule(rule.requiredType())
                        .status("PASS")
                        .currentVersion(targetEcu.versionString())
                        .requiredVersion(rule.requiredVersionString())
                        .build();
                report.add(result);
                recordHistory(vehicleId, packageId, rule.requiredType(), result, null);
            } else {
                RuleResult result = fail(rule.requiredType(), targetEcu.versionString(), rule.requiredVersionString(),
                        FailureReason.VERSION_BELOW_REQUIRED,
                        String.format("버전 미달 (현재: v%s < 요구: v%s)",
                                targetEcu.versionString(), rule.requiredVersionString()));
                report.add(result);
                recordHistory(vehicleId, packageId, rule.requiredType(), result, FailureReason.VERSION_BELOW_REQUIRED);
            }
        }

        boolean allPassed = report.stream().allMatch(result -> "PASS".equals(result.getStatus()));
        log.info("[OTA 검증] vehicle={}, package={}, result={}", vehicleId, packageId, allPassed);

        return CheckResponse.builder()
                .vehicleId(vehicleId)
                .packageId(packageId)
                .available(allPassed)
                .details(report)
                .build();
    }

    private RuleResult fail(
            String rule,
            String currentVersion,
            String requiredVersion,
            FailureReason reasonCode,
            String reason
    ) {
        return RuleResult.builder()
                .rule(rule)
                .status("FAIL")
                .currentVersion(currentVersion)
                .requiredVersion(requiredVersion)
                .reasonCode(reasonCode.name())
                .reason(reason)
                .build();
    }

    private boolean isStale(EcuInventoryResponse ecu) {
        if (ecu.lastReportedAt() == null) {
            return true;
        }
        return ecu.lastReportedAt().isBefore(Instant.now().minus(Duration.ofMinutes(maxInventoryAgeMinutes)));
    }

    private void recordHistory(
            String vehicleId,
            String packageId,
            String ecuType,
            RuleResult result,
            FailureReason reasonCode
    ) {
        validationHistoryRepository.save(ValidationHistory.builder()
                .vehicleId(vehicleId)
                .packageId(packageId)
                .ecuType(ecuType)
                .status("PASS".equals(result.getStatus()) ? ValidationStatus.PASS : ValidationStatus.FAIL)
                .reasonCode(reasonCode)
                .reason(result.getReason())
                .currentVersion(result.getCurrentVersion())
                .requiredVersion(result.getRequiredVersion())
                .createdAt(Instant.now())
                .build());
    }
}
