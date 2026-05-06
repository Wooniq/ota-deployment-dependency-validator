package com.ota.campaign.dto;

import com.ota.campaign.domain.CampaignTarget;

public record CampaignTargetResponse(
        Long id,
        Long campaignId,
        String vehicleId,
        String status
) {
    public static CampaignTargetResponse from(CampaignTarget target) {
        return new CampaignTargetResponse(
                target.getId(),
                target.getCampaign().getId(),
                target.getVehicleId(),
                target.getStatus().name()
        );
    }
}
