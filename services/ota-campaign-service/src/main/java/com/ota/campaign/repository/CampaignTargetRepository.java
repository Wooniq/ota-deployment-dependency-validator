package com.ota.campaign.repository;

import com.ota.campaign.domain.CampaignTarget;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface CampaignTargetRepository extends JpaRepository<CampaignTarget, Long> {

    List<CampaignTarget> findByCampaignId(Long campaignId);

    boolean existsByCampaignIdAndVehicleId(Long campaignId, String vehicleId);
}
