package com.ota.control.domain;

import jakarta.persistence.*;
import lombok.*;
import java.time.LocalDateTime;

@Entity
@Table(name = "campaigns")
@Getter @Setter
@NoArgsConstructor
@AllArgsConstructor
@Builder
public class Campaign {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(name = "campaign_name", nullable = false, length = 100)
    private String campaignName;

    @ManyToOne(fetch = FetchType.LAZY)
    @JoinColumn(name = "package_id", nullable = false)
    private UpdatePackage updatePackage;

    @Enumerated(EnumType.STRING)
    @Column(nullable = false, length = 20)
    private CampaignStatus status;

    @Column(name = "created_at", nullable = false)
    private LocalDateTime createdAt;

    @Column(name = "started_at")
    private LocalDateTime startedAt;

    @Column(name = "finished_at")
    private LocalDateTime finishedAt;

    @Column(name = "total_vehicles")
    private Integer totalVehicles;

    @Column(name = "completed_count")
    private Integer completedCount;

    @Column(name = "failed_count")
    private Integer failedCount;

    @PrePersist
    protected void onCreate() {
        this.createdAt = LocalDateTime.now();
        this.completedCount = 0;
        this.failedCount = 0;
    }

    public enum CampaignStatus {
        CREATED,
        VALIDATING,
        IN_PROGRESS,
        PAUSED,
        COMPLETED,
        ABORTED
    }
}
