package com.ota.control.repository;

import com.ota.control.domain.Vehicle;
import com.ota.control.domain.Vehicle.VehicleStatus;
import org.springframework.data.jpa.repository.JpaRepository;
import java.util.List;

public interface VehicleRepository extends JpaRepository<Vehicle, String> {

    List<Vehicle> findByStatus(VehicleStatus status);

    List<Vehicle> findByModelName(String modelName);
}
