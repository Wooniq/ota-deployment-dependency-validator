package com.ota.campaign.dto;

import java.util.List;

public record CampaignStartRequest(
        List<String> targetVehicleIds
) {
}
