package com.ota.control.controller;

import com.ota.control.domain.Ecu;
import com.ota.control.domain.Vehicle;
import com.ota.control.repository.EcuRepository;
import com.ota.control.repository.VehicleRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.List;

/**
 * 차량 & ECU 관리 API
 * 과제 2번: 데이터베이스에 데이터를 삽입하고 조회하는 애플리케이션
 */
@RestController
@RequestMapping("/api/vehicles")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class VehicleController {

    private final VehicleRepository vehicleRepository;
    private final EcuRepository ecuRepository;

    // ── Vehicle CRUD ──

    @GetMapping
    public List<Vehicle> getAllVehicles() {
        return vehicleRepository.findAll();
    }

    @GetMapping("/{vehicleId}")
    public ResponseEntity<Vehicle> getVehicle(@PathVariable String vehicleId) {
        return vehicleRepository.findById(vehicleId)
                .map(ResponseEntity::ok)
                .orElse(ResponseEntity.notFound().build());
    }

    @PostMapping
    public Vehicle createVehicle(@RequestBody Vehicle vehicle) {
        return vehicleRepository.save(vehicle);
    }

    @PutMapping("/{vehicleId}/status")
    public ResponseEntity<Vehicle> updateStatus(
            @PathVariable String vehicleId,
            @RequestParam Vehicle.VehicleStatus status) {
        return vehicleRepository.findById(vehicleId)
                .map(v -> {
                    v.setStatus(status);
                    return ResponseEntity.ok(vehicleRepository.save(v));
                })
                .orElse(ResponseEntity.notFound().build());
    }

    @DeleteMapping("/{vehicleId}")
    public ResponseEntity<Void> deleteVehicle(@PathVariable String vehicleId) {
        if (vehicleRepository.existsById(vehicleId)) {
            vehicleRepository.deleteById(vehicleId);
            return ResponseEntity.noContent().build();
        }
        return ResponseEntity.notFound().build();
    }

    // ── ECU 조회 ──

    @GetMapping("/{vehicleId}/ecus")
    public List<Ecu> getVehicleEcus(@PathVariable String vehicleId) {
        return ecuRepository.findByVehicleId(vehicleId);
    }

    @PostMapping("/{vehicleId}/ecus")
    public Ecu addEcu(@PathVariable String vehicleId, @RequestBody Ecu ecu) {
        ecu.setVehicleId(vehicleId);
        return ecuRepository.save(ecu);
    }
}
