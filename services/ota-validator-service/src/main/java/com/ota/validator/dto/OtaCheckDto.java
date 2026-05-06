package com.ota.validator.dto;

import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;

import java.util.List;

public class OtaCheckDto {

    @Getter
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CheckRequest {
        private String vehicleId;
        private String packageId;
    }

    @Getter
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class CheckResponse {
        private String vehicleId;
        private String packageId;
        private boolean available;
        private List<RuleResult> details;
    }

    @Getter
    @Builder
    @NoArgsConstructor
    @AllArgsConstructor
    public static class RuleResult {
        private String rule;
        private String status;
        private String currentVersion;
        private String requiredVersion;
        private String reasonCode;
        private String reason;
    }
}
