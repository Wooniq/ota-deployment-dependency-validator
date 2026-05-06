package com.ota.campaign.dto;

import com.ota.campaign.domain.UpdatePackage;

public record UpdatePackageResponse(
        String packageId,
        String targetEcuType,
        Integer major,
        Integer minor,
        Integer patch,
        String version,
        String s3Key
) {
    public static UpdatePackageResponse from(UpdatePackage updatePackage) {
        return new UpdatePackageResponse(
                updatePackage.getPackageId(),
                updatePackage.getTargetEcuType(),
                updatePackage.getMajor(),
                updatePackage.getMinor(),
                updatePackage.getPatch(),
                updatePackage.getVersionString(),
                updatePackage.getS3Key()
        );
    }
}
