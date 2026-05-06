package com.ota.validator.repository;

import com.ota.validator.domain.ValidationHistory;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface ValidationHistoryRepository extends JpaRepository<ValidationHistory, Long> {

    List<ValidationHistory> findByVehicleIdOrderByCreatedAtDesc(String vehicleId);
}
