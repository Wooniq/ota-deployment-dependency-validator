package com.ota.control.config;

import com.ota.control.domain.*;
import com.ota.control.repository.*;
import lombok.RequiredArgsConstructor;
import lombok.extern.slf4j.Slf4j;
import org.springframework.boot.CommandLineRunner;
import org.springframework.context.annotation.Profile;
import org.springframework.stereotype.Component;

import java.util.List;

/**
 * 개발용 샘플 데이터 초기화
 * ← Python test_connection.py의 populate_data() 포팅
 *
 * local 프로필에서만 실행됨
 */
@Component
@Profile("local")
@RequiredArgsConstructor
@Slf4j
public class DataInitializer implements CommandLineRunner {

    private final VehicleRepository vehicleRepo;
    private final EcuRepository ecuRepo;
    private final UpdatePackageRepository packageRepo;
    private final DependencyRuleRepository ruleRepo;

    @Override
    public void run(String... args) {
        if (vehicleRepo.count() > 0) {
            log.info("[Init] 데이터 이미 존재, 초기화 스킵");
            return;
        }

        // 1. 테스트 차량 데이터
        Vehicle v001 = vehicleRepo.save(Vehicle.builder()
                .vehicleId("V001").modelName("IONIQ6").status(Vehicle.VehicleStatus.READY).build());
        Vehicle v002 = vehicleRepo.save(Vehicle.builder()
                .vehicleId("V002").modelName("GV80").status(Vehicle.VehicleStatus.READY).build());

        // 2. ECU 현황 (V002는 BCM 버전이 낮음 → 검증 실패 유도용)
        ecuRepo.saveAll(List.of(
                Ecu.builder().vehicleId("V001").ecuType("BMS").major(2).minor(0).patch(0).build(),
                Ecu.builder().vehicleId("V001").ecuType("BCM").major(1).minor(5).patch(0).build(),
                Ecu.builder().vehicleId("V002").ecuType("BMS").major(2).minor(0).patch(0).build(),
                Ecu.builder().vehicleId("V002").ecuType("BCM").major(1).minor(0).patch(0).build()
        ));

        // 3. 업데이트 패키지
        UpdatePackage pkg = packageRepo.save(UpdatePackage.builder()
                .packageId("PKG_BMS_30")
                .targetEcuType("BMS")
                .major(3).minor(0).patch(0)
                .s3Key("firmware/BMS/v3.0.0.bin")
                .build());

        // 4. 의존성 규칙: BMS 3.0 설치하려면 BCM ≥ 1.2.0
        ruleRepo.save(DependencyRule.builder()
                .updatePackage(pkg)
                .requiredType("BCM")
                .minMajor(1).minMinor(2).minPatch(0)
                .build());

        log.info("[Init] 샘플 데이터 삽입 완료 (V001=IONIQ6, V002=GV80)");
    }
}
