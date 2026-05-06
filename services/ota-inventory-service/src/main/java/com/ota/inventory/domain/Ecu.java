package com.ota.inventory.domain;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;
import jakarta.persistence.UniqueConstraint;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

import java.time.Instant;

@Entity
@Table(
        name = "ecus",
        uniqueConstraints = @UniqueConstraint(
                name = "uk_ecu_vehicle_type",
                columnNames = {"vehicle_id", "ecu_type"}
        )
)
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Ecu {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "vehicle_id", nullable = false, length = 20)
    private String vehicleId;

    @Column(name = "ecu_type", nullable = false, length = 20)
    private String ecuType;

    @Column(nullable = false)
    private Integer major;

    @Column(nullable = false)
    private Integer minor;

    @Column(nullable = false)
    private Integer patch;

    @Column(name = "last_reported_at", nullable = false)
    private Instant lastReportedAt;

    public String getVersionString() {
        return major + "." + minor + "." + patch;
    }
}
