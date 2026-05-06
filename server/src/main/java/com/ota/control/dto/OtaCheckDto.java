package com.ota.control.dto;

import jakarta.validation.constraints.NotBlank;
import jakarta.validation.constraints.NotNull;
import lombok.*;
import java.util.List;

public class OtaCheckDto {

    /**
     * 업데이트 검증 요청 ← Python CheckRequest 모델 포팅
     */
    @Getter @Setter
    @NoArgsConstructor @AllArgsConstructor
    public static class CheckRequest {
        @NotBlank
        private String vehicleId;
        @NotBlank
        private String packageId;
    }

    /**
     * 검증 결과 응답 ← Python validate_ota() 반환값 포팅
     */
    @Getter @Setter
    @NoArgsConstructor @AllArgsConstructor @Builder
    public static class CheckResponse {
        private String vehicleId;
        private String packageId;
        private boolean available;
        private List<RuleResult> details;
    }

    /**
     * 개별 규칙 검증 결과
     */
    @Getter @Setter
    @NoArgsConstructor @AllArgsConstructor @Builder
    public static class RuleResult {
        private String rule;            // ECU 타입
        private String status;          // PASS / FAIL
        private String currentVersion;
        private String requiredVersion;
        private String reasonCode;      // 실패 사유 코드 (nullable)
        private String reason;          // 실패 사유 (nullable)
    }
}
