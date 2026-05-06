package com.ota.campaign.dto;

import java.util.List;

public record CreateCampaignRequest(
        String packageId,
        String campaignName,
        Integer totalVehicles,
        List<String> targetVehicleIds
) {
}
