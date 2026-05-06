package com.ota.inventory.repository;

import com.ota.inventory.domain.Ecu;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;
import java.util.Optional;

public interface EcuRepository extends JpaRepository<Ecu, Long> {

    List<Ecu> findByVehicleId(String vehicleId);

    Optional<Ecu> findByVehicleIdAndEcuType(String vehicleId, String ecuType);
}
