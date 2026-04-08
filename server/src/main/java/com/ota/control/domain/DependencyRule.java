package com.ota.control.domain;

import jakarta.persistence.*;
import lombok.*;

@Entity
@Table(name = "dependency_rules")
@Getter @Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class DependencyRule {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    @Column(name = "rule_id")
    private Long ruleId;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "package_id", nullable = false)
    private UpdatePackage updatePackage;

    /** 의존 대상 ECU 타입 (예: BCM) */
    @Column(name = "required_type", nullable = false, length = 20)
    private String requiredType;

    @Column(name = "min_major", nullable = false)
    private Integer minMajor;

    @Column(name = "min_minor", nullable = false)
    private Integer minMinor;

    @Column(name = "min_patch", nullable = false)
    private Integer minPatch;

    public String getRequiredVersionString() {
        return minMajor + "." + minMinor + "." + minPatch;
    }
}
