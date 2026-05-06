package com.ota.campaign.domain;

import jakarta.persistence.CascadeType;
import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.FetchType;
import jakarta.persistence.Id;
import jakarta.persistence.OneToMany;
import jakarta.persistence.Table;
import lombok.AllArgsConstructor;
import lombok.Builder;
import lombok.Getter;
import lombok.NoArgsConstructor;
import lombok.Setter;

import java.util.List;

@Entity
@Table(name = "update_packages")
@Getter
@Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class UpdatePackage {

    @Id
    @Column(name = "package_id", length = 50)
    private String packageId;

    @Column(name = "target_ecu_type", nullable = false, length = 20)
    private String targetEcuType;

    @Column(nullable = false)
    private Integer major;

    @Column(nullable = false)
    private Integer minor;

    @Column(nullable = false)
    private Integer patch;

    @Column(name = "s3_key", length = 500)
    private String s3Key;

    @OneToMany(mappedBy = "updatePackage", cascade = CascadeType.ALL, fetch = FetchType.LAZY)
    private List<DependencyRule> dependencyRules;

    public String getVersionString() {
        return major + "." + minor + "." + patch;
    }
}
