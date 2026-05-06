package com.ota.campaign.domain;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.FetchType;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.JoinColumn;
import jakarta.persistence.ManyToOne;
import jakarta.persistence.Table;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

@Entity
@Table(name = "dependency_rules")
@Getter
@Setter
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
