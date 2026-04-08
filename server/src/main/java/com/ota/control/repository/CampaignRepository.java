package com.ota.control.repository;

import com.ota.control.domain.Campaign;
import com.ota.control.domain.Campaign.CampaignStatus;
import org.springframework.data.jpa.repository.JpaRepository;
import java.util.List;

public interface CampaignRepository extends JpaRepository<Campaign, Long> {

    List<Campaign> findByStatus(CampaignStatus status);
}
