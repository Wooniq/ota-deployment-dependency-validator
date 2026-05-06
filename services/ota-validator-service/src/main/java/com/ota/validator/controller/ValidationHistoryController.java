package com.ota.validator.controller;

import com.ota.validator.domain.ValidationHistory;
import com.ota.validator.repository.ValidationHistoryRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/api/validation-histories")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class ValidationHistoryController {

    private final ValidationHistoryRepository validationHistoryRepository;

    @GetMapping
    public List<ValidationHistory> getHistories() {
        return validationHistoryRepository.findAll();
    }

    @GetMapping("/vehicles/{vehicleId}")
    public List<ValidationHistory> getVehicleHistories(@PathVariable String vehicleId) {
        return validationHistoryRepository.findByVehicleIdOrderByCreatedAtDesc(vehicleId);
    }
}
