package com.ota.control.domain;

import jakarta.persistence.*;
import lombok.*;

@Entity
@Table(name = "ecus")
@Getter @Setter
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
    private String ecuType;  // BMS, BCM, ADAS 등

    @Column(nullable = false)
    private Integer major;

    @Column(nullable = false)
    private Integer minor;

    @Column(nullable = false)
    private Integer patch;

    /**
     * Semantic Version 문자열 반환 (예: "2.1.0")
     */
    public String getVersionString() {
        return major + "." + minor + "." + patch;
    }

    /**
     * 버전 비교: 현재 버전이 요구 버전 이상인지 확인
     * Python validator.py의 is_compatible(current, required) 포팅
     */
    public boolean isCompatibleWith(int reqMajor, int reqMinor, int reqPatch) {
        if (this.major != reqMajor) return this.major > reqMajor;
        if (this.minor != reqMinor) return this.minor > reqMinor;
        return this.patch >= reqPatch;
    }
}
