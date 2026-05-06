package com.ota.validator.dto;

public record DependencyRuleResponse(
        Long ruleId,
        String packageId,
        String requiredType,
        Integer minMajor,
        Integer minMinor,
        Integer minPatch,
        String requiredVersion
) {
    public String requiredVersionString() {
        if (requiredVersion != null) {
            return requiredVersion;
        }
        return minMajor + "." + minMinor + "." + minPatch;
    }
}
