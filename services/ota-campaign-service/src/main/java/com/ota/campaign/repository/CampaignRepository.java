package com.ota.campaign.repository;

import com.ota.campaign.domain.Campaign;
import com.ota.campaign.domain.Campaign.CampaignStatus;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface CampaignRepository extends JpaRepository<Campaign, Long> {

    List<Campaign> findByStatus(CampaignStatus status);
}
