package com.ota.inventory.controller;

import com.ota.inventory.dto.EcuInventoryResponse;
import com.ota.inventory.repository.EcuRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/internal")
@RequiredArgsConstructor
public class InternalInventoryController {

    private final EcuRepository ecuRepository;

    @GetMapping("/vehicles/{vehicleId}/ecus")
    public List<EcuInventoryResponse> getEcus(@PathVariable String vehicleId) {
        return ecuRepository.findByVehicleId(vehicleId).stream()
                .map(EcuInventoryResponse::from)
                .toList();
    }
}
