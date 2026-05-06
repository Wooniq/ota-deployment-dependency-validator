package com.ota.campaign.dto;

import com.ota.campaign.domain.DependencyRule;

public record DependencyRuleResponse(
        Long ruleId,
        String packageId,
        String requiredType,
        Integer minMajor,
        Integer minMinor,
        Integer minPatch,
        String requiredVersion
) {
    public static DependencyRuleResponse from(DependencyRule rule) {
        return new DependencyRuleResponse(
                rule.getRuleId(),
                rule.getUpdatePackage().getPackageId(),
                rule.getRequiredType(),
                rule.getMinMajor(),
                rule.getMinMinor(),
                rule.getMinPatch(),
                rule.getRequiredVersionString()
        );
    }
}
