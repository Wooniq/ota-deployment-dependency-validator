package com.ota.control.domain;

import jakarta.persistence.*;
import lombok.*;
import java.util.List;

@Entity
@Table(name = "update_packages")
@Getter @Setter
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

    /** S3에 업로드된 펌웨어 파일 키 */
    @Column(name = "s3_key", length = 500)
    private String s3Key;

    @OneToMany(mappedBy = "updatePackage", cascade = CascadeType.ALL, fetch = FetchType.LAZY)
    private List<DependencyRule> dependencyRules;

    public String getVersionString() {
        return major + "." + minor + "." + patch;
    }
}
