package com.ota.campaign.repository;

import com.ota.campaign.domain.DependencyRule;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface DependencyRuleRepository extends JpaRepository<DependencyRule, Long> {

    List<DependencyRule> findByUpdatePackagePackageId(String packageId);
}
