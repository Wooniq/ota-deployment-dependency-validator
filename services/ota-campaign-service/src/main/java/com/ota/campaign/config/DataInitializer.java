package com.ota.campaign.config;

import com.ota.campaign.domain.DependencyRule;
import com.ota.campaign.domain.UpdatePackage;
import com.ota.campaign.repository.DependencyRuleRepository;
import com.ota.campaign.repository.UpdatePackageRepository;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Component;

@Component
@Profile("local")
@RequiredArgsConstructor
@Slf4j
public class DataInitializer implements CommandLineRunner {

    private final UpdatePackageRepository packageRepository;
    private final DependencyRuleRepository ruleRepository;

    @Override
    public void run(String... args) {
        if (packageRepository.existsById("PKG_BMS_30")) {
            log.info("[Init] Campaign sample data already exists");
            return;
        }

        UpdatePackage updatePackage = packageRepository.save(UpdatePackage.builder()
                .packageId("PKG_BMS_30")
                .targetEcuType("BMS")
                .major(3).minor(0).patch(0)
                .s3Key("firmware/BMS/v3.0.0.bin")
                .build());

        ruleRepository.save(DependencyRule.builder()
                .updatePackage(updatePackage)
                .requiredType("BCM")
                .minMajor(1).minMinor(2).minPatch(0)
                .build());

        log.info("[Init] Campaign sample data inserted: packageId=PKG_BMS_30");
    }
}
