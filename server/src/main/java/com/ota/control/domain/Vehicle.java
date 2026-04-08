package com.ota.control.domain;

import jakarta.persistence.*;
import lombok.*;

@Entity
@Table(name = "vehicles")
@Getter @Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Vehicle {

    @Id
    @Column(name = "vehicle_id", length = 20)
    private String vehicleId;

    @Column(name = "model_name", nullable = false, length = 50)
    private String modelName;

    @Enumerated(EnumType.STRING)
    @Column(name = "status", nullable = false, length = 20)
    private VehicleStatus status;

    public enum VehicleStatus {
        READY,          // 업데이트 대기
        DOWNLOADING,    // 다운로드 중
        INSTALLING,     // 설치 중
        COMPLETED,      // 완료
        FAILED,         // 실패
        ROLLBACK        // 롤백 수행
    }
}
