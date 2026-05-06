package com.ota.campaign.controller;

import com.ota.campaign.domain.Campaign;
import com.ota.campaign.domain.Campaign.CampaignStatus;
import com.ota.campaign.domain.CampaignTarget;
import com.ota.campaign.dto.CampaignEvent;
import com.ota.campaign.dto.CampaignStartRequest;
import com.ota.campaign.dto.CampaignTargetResponse;
import com.ota.campaign.dto.CreateCampaignRequest;
import com.ota.campaign.messaging.CampaignEventPublisher;
import com.ota.campaign.repository.CampaignRepository;
import com.ota.campaign.repository.CampaignTargetRepository;
import com.ota.campaign.repository.UpdatePackageRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.time.Instant;
import java.time.LocalDateTime;
import java.util.List;

@RestController
@RequestMapping("/api/campaigns")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class CampaignController {

    private final CampaignRepository campaignRepository;
    private final UpdatePackageRepository packageRepository;
    private final CampaignTargetRepository targetRepository;
    private final CampaignEventPublisher campaignEventPublisher;

    @GetMapping
    public List<Campaign> getAllCampaigns() {
        return campaignRepository.findAll();
    }

    @GetMapping("/{id}")
    public ResponseEntity<Campaign> getCampaign(@PathVariable Long id) {
        return campaignRepository.findById(id)
                .map(ResponseEntity::ok)
                .orElse(ResponseEntity.notFound().build());
    }

    @PostMapping
    public ResponseEntity<Campaign> createCampaign(@RequestBody CreateCampaignRequest request) {
        return packageRepository.findById(request.packageId())
                .map(pkg -> {
                    Campaign campaign = Campaign.builder()
                            .campaignName(request.campaignName())
                            .updatePackage(pkg)
                            .status(CampaignStatus.CREATED)
                            .totalVehicles(resolveTotalVehicles(request))
                            .build();
                    Campaign saved = campaignRepository.save(campaign);
                    saveTargets(saved, request.targetVehicleIds());
                    return ResponseEntity.ok(saved);
                })
                .orElse(ResponseEntity.badRequest().build());
    }

    @GetMapping("/{id}/targets")
    public ResponseEntity<List<CampaignTargetResponse>> getTargets(@PathVariable Long id) {
        if (!campaignRepository.existsById(id)) {
            return ResponseEntity.notFound().build();
        }
        List<CampaignTargetResponse> targets = targetRepository.findByCampaignId(id).stream()
                .map(CampaignTargetResponse::from)
                .toList();
        return ResponseEntity.ok(targets);
    }

    @PutMapping("/{id}/start")
    public ResponseEntity<Campaign> startCampaign(
            @PathVariable Long id,
            @RequestBody(required = false) CampaignStartRequest request
    ) {
        return campaignRepository.findById(id)
                .map(campaign -> {
                    if (request != null && request.targetVehicleIds() != null) {
                        saveTargets(campaign, request.targetVehicleIds());
                    }

                    campaign.setStatus(CampaignStatus.IN_PROGRESS);
                    campaign.setStartedAt(LocalDateTime.now());
                    Campaign saved = campaignRepository.save(campaign);

                    List<String> targetVehicleIds = targetRepository.findByCampaignId(saved.getId()).stream()
                            .map(CampaignTarget::getVehicleId)
                            .toList();
                    campaignEventPublisher.publishStarted(new CampaignEvent(
                            saved.getId(),
                            saved.getUpdatePackage().getPackageId(),
                            targetVehicleIds,
                            Instant.now()
                    ));

                    return ResponseEntity.ok(saved);
                })
                .orElse(ResponseEntity.notFound().build());
    }

    @PutMapping("/{id}/abort")
    public ResponseEntity<Campaign> abortCampaign(@PathVariable Long id) {
        return campaignRepository.findById(id)
                .map(campaign -> {
                    campaign.setStatus(CampaignStatus.ABORTED);
                    campaign.setFinishedAt(LocalDateTime.now());
                    return ResponseEntity.ok(campaignRepository.save(campaign));
                })
                .orElse(ResponseEntity.notFound().build());
    }

    private Integer resolveTotalVehicles(CreateCampaignRequest request) {
        if (request.totalVehicles() != null) {
            return request.totalVehicles();
        }
        return request.targetVehicleIds() != null ? request.targetVehicleIds().size() : 0;
    }

    private void saveTargets(Campaign campaign, List<String> targetVehicleIds) {
        if (targetVehicleIds == null || targetVehicleIds.isEmpty()) {
            return;
        }

        List<CampaignTarget> targets = targetVehicleIds.stream()
                .distinct()
                .filter(vehicleId -> !targetRepository.existsByCampaignIdAndVehicleId(campaign.getId(), vehicleId))
                .map(vehicleId -> CampaignTarget.builder()
                        .campaign(campaign)
                        .vehicleId(vehicleId)
                        .status(CampaignTarget.TargetStatus.PENDING)
                        .build())
                .toList();

        if (!targets.isEmpty()) {
            targetRepository.saveAll(targets);
        }
    }
}
