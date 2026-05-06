package com.ota.inventory.controller;

import com.ota.inventory.domain.Ecu;
import com.ota.inventory.domain.Vehicle;
import com.ota.inventory.repository.EcuRepository;
import com.ota.inventory.repository.VehicleRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.time.Instant;
import java.util.List;

@RestController
@RequestMapping("/api/vehicles")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class VehicleController {

    private final VehicleRepository vehicleRepository;
    private final EcuRepository ecuRepository;

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
            @RequestParam Vehicle.VehicleStatus status
    ) {
        return vehicleRepository.findById(vehicleId)
                .map(vehicle -> {
                    vehicle.setStatus(status);
                    return ResponseEntity.ok(vehicleRepository.save(vehicle));
                })
                .orElse(ResponseEntity.notFound().build());
    }

    @GetMapping("/{vehicleId}/ecus")
    public List<Ecu> getVehicleEcus(@PathVariable String vehicleId) {
        return ecuRepository.findByVehicleId(vehicleId);
    }

    @PostMapping("/{vehicleId}/ecus")
    public Ecu addEcu(@PathVariable String vehicleId, @RequestBody Ecu ecu) {
        ecu.setVehicleId(vehicleId);
        if (ecu.getLastReportedAt() == null) {
            ecu.setLastReportedAt(Instant.now());
        }
        return ecuRepository.findByVehicleIdAndEcuType(vehicleId, ecu.getEcuType())
                .map(existing -> {
                    existing.setMajor(ecu.getMajor());
                    existing.setMinor(ecu.getMinor());
                    existing.setPatch(ecu.getPatch());
                    existing.setLastReportedAt(ecu.getLastReportedAt());
                    return ecuRepository.save(existing);
                })
                .orElseGet(() -> ecuRepository.save(ecu));
    }
}
