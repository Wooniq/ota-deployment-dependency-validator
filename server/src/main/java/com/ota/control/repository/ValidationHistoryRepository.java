package com.ota.control.repository;

import com.ota.control.domain.ValidationHistory;
import org.springframework.data.jpa.repository.JpaRepository;

public interface ValidationHistoryRepository extends JpaRepository<ValidationHistory, Long> {
}
