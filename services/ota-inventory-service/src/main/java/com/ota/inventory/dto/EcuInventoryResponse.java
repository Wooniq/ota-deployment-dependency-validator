package com.ota.inventory.dto;

import com.ota.inventory.domain.Ecu;

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
    public static EcuInventoryResponse from(Ecu ecu) {
        return new EcuInventoryResponse(
                ecu.getVehicleId(),
                ecu.getEcuType(),
                ecu.getMajor(),
                ecu.getMinor(),
                ecu.getPatch(),
                ecu.getVersionString(),
                ecu.getLastReportedAt()
        );
    }
}
