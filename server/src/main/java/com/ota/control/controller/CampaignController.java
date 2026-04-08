package com.ota.control.controller;

import com.ota.control.domain.Campaign;
import com.ota.control.domain.Campaign.CampaignStatus;
import com.ota.control.repository.CampaignRepository;
import com.ota.control.repository.UpdatePackageRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.time.LocalDateTime;
import java.util.List;
import java.util.Map;

/**
 * 배포 캠페인 관리 API
 */
@RestController
@RequestMapping("/api/campaigns")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class CampaignController {

    private final CampaignRepository campaignRepository;
    private final UpdatePackageRepository packageRepository;

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
    public ResponseEntity<Campaign> createCampaign(@RequestBody Map<String, Object> body) {
        String packageId = (String) body.get("packageId");
        String name = (String) body.get("campaignName");
        Integer totalVehicles = (Integer) body.get("totalVehicles");

        return packageRepository.findById(packageId)
                .map(pkg -> {
                    Campaign campaign = Campaign.builder()
                            .campaignName(name)
                            .updatePackage(pkg)
                            .status(CampaignStatus.CREATED)
                            .totalVehicles(totalVehicles != null ? totalVehicles : 0)
                            .build();
                    return ResponseEntity.ok(campaignRepository.save(campaign));
                })
                .orElse(ResponseEntity.badRequest().build());
    }

    @PutMapping("/{id}/start")
    public ResponseEntity<Campaign> startCampaign(@PathVariable Long id) {
        return campaignRepository.findById(id)
                .map(c -> {
                    c.setStatus(CampaignStatus.IN_PROGRESS);
                    c.setStartedAt(LocalDateTime.now());
                    return ResponseEntity.ok(campaignRepository.save(c));
                })
                .orElse(ResponseEntity.notFound().build());
    }

    @PutMapping("/{id}/abort")
    public ResponseEntity<Campaign> abortCampaign(@PathVariable Long id) {
        return campaignRepository.findById(id)
                .map(c -> {
                    c.setStatus(CampaignStatus.ABORTED);
                    c.setFinishedAt(LocalDateTime.now());
                    return ResponseEntity.ok(campaignRepository.save(c));
                })
                .orElse(ResponseEntity.notFound().build());
    }
}
