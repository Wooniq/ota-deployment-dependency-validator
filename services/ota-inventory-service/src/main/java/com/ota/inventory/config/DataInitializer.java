package com.ota.inventory.config;

import com.ota.inventory.domain.Ecu;
import com.ota.inventory.domain.Vehicle;
import com.ota.inventory.repository.EcuRepository;
import com.ota.inventory.repository.VehicleRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Component;

import java.time.Instant;
import java.util.List;

@Component
@Profile("local")
@RequiredArgsConstructor
@Slf4j
public class DataInitializer implements CommandLineRunner {

    private final VehicleRepository vehicleRepository;
    private final EcuRepository ecuRepository;

    @Override
    public void run(String... args) {
        if (vehicleRepository.count() > 0) {
            log.info("[Init] Inventory sample data already exists");
            return;
        }

        vehicleRepository.saveAll(List.of(
                Vehicle.builder().vehicleId("V001").modelName("IONIQ6").status(Vehicle.VehicleStatus.READY).build(),
                Vehicle.builder().vehicleId("V002").modelName("GV80").status(Vehicle.VehicleStatus.READY).build()
        ));

        Instant now = Instant.now();
        ecuRepository.saveAll(List.of(
                Ecu.builder().vehicleId("V001").ecuType("BMS").major(2).minor(0).patch(0).lastReportedAt(now).build(),
                Ecu.builder().vehicleId("V001").ecuType("BCM").major(1).minor(5).patch(0).lastReportedAt(now).build(),
                Ecu.builder().vehicleId("V002").ecuType("BMS").major(2).minor(0).patch(0).lastReportedAt(now).build(),
                Ecu.builder().vehicleId("V002").ecuType("BCM").major(1).minor(0).patch(0).lastReportedAt(now).build()
        ));

        log.info("[Init] Inventory sample data inserted");
    }
}
