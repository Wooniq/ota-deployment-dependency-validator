package com.ota.validator.dto;

import java.time.Instant;

public record EcuInventoryResponse(
        String vehicleId,
        String ecuType,
        Integer major,
        Integer minor,
        Integer patch,
        String version,
        Instant lastReportedAt
) {
    public String versionString() {
        if (version != null) {
            return version;
        }
        return major + "." + minor + "." + patch;
    }

    public boolean isCompatibleWith(int requiredMajor, int requiredMinor, int requiredPatch) {
        if (!major.equals(requiredMajor)) {
            return major > requiredMajor;
        }
        if (!minor.equals(requiredMinor)) {
            return minor > requiredMinor;
        }
        return patch >= requiredPatch;
    }
}
