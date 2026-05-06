package com.ota.campaign.dto;

import java.time.Instant;
import java.util.List;

public record CampaignEvent(
        Long campaignId,
        String packageId,
        List<String> targetVehicleIds,
        Instant startedAt
) {
}
