package com.ota.validator.controller;

import com.ota.validator.dto.OtaCheckDto.CheckRequest;
import com.ota.validator.dto.OtaCheckDto.CheckResponse;
import com.ota.validator.service.OtaValidatorService;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/ota")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class OtaCheckController {

    private final OtaValidatorService validatorService;

    @PostMapping("/check-update")
    public CheckResponse checkUpdate(@RequestBody CheckRequest request) {
        return validatorService.validateUpdate(request.getVehicleId(), request.getPackageId());
    }
}
