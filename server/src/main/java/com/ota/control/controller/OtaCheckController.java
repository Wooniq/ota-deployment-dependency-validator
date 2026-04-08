package com.ota.control.controller;

import com.ota.control.dto.OtaCheckDto.*;
import com.ota.control.service.OtaValidatorService;
import jakarta.validation.Valid;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

/**
 * OTA 업데이트 검증 API
 * ← Python FastAPI @app.post("/check-update") 포팅
 */
@RestController
@RequestMapping("/api/ota")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class OtaCheckController {

    private final OtaValidatorService validatorService;

    /**
     * 차량에 대한 패키지 업데이트 가능 여부 검증
     * POST /api/ota/check-update
     */
    @PostMapping("/check-update")
    public CheckResponse checkUpdate(@Valid @RequestBody CheckRequest request) {
        return validatorService.validateUpdate(request.getVehicleId(), request.getPackageId());
    }

    /**
     * 헬스체크
     * GET /api/ota
     */
    @GetMapping
    public java.util.Map<String, String> healthCheck() {
        return java.util.Map.of("message", "OTA Validator API 가동 중 ...");
    }
}
