package com.ota.campaign.controller;

import com.ota.campaign.dto.DependencyRuleResponse;
import com.ota.campaign.dto.UpdatePackageResponse;
import com.ota.campaign.repository.DependencyRuleRepository;
import com.ota.campaign.repository.UpdatePackageRepository;
import lombok.RequiredArgsConstructor;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import java.util.List;

@RestController
@RequestMapping("/internal/packages")
@RequiredArgsConstructor
public class InternalPackageController {

    private final UpdatePackageRepository packageRepository;
    private final DependencyRuleRepository ruleRepository;

    @GetMapping("/{packageId}")
    public ResponseEntity<UpdatePackageResponse> getPackage(@PathVariable String packageId) {
        return packageRepository.findById(packageId)
                .map(UpdatePackageResponse::from)
                .map(ResponseEntity::ok)
                .orElse(ResponseEntity.notFound().build());
    }

    @GetMapping("/{packageId}/rules")
    public List<DependencyRuleResponse> getRules(@PathVariable String packageId) {
        return ruleRepository.findByUpdatePackagePackageId(packageId).stream()
                .map(DependencyRuleResponse::from)
                .toList();
    }
}
