package com.ota.control.domain;

import jakarta.persistence.*;
import lombok.*;
import java.time.Instant;

@Entity
@Table(name = "validation_histories")
@Getter @Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class ValidationHistory {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "vehicle_id", nullable = false, length = 20)
    private String vehicleId;

    @Column(name = "package_id", nullable = false, length = 50)
    private String packageId;

    @Column(name = "ecu_type", nullable = false, length = 20)
    private String ecuType;

    @Enumerated(EnumType.STRING)
    @Column(name = "status", nullable = false, length = 10)
    private ValidationStatus status;

    @Enumerated(EnumType.STRING)
    @Column(name = "reason_code", length = 40)
    private FailureReason reasonCode;

    @Column(name = "reason", length = 500)
    private String reason;

    @Column(name = "current_version", length = 30)
    private String currentVersion;

    @Column(name = "required_version", length = 30)
    private String requiredVersion;

    @Column(name = "created_at", nullable = false)
    private Instant createdAt;

    public enum ValidationStatus {
        PASS,
        FAIL
    }

    public enum FailureReason {
        MISSING_REQUIRED_ECU,
        VERSION_BELOW_REQUIRED,
        STALE_INVENTORY
    }
}
