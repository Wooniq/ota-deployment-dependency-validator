package com.ota.firmware.controller;

import com.ota.firmware.service.S3FirmwareService;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.CrossOrigin;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.Map;

@RestController
@RequestMapping("/api/firmware")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class FirmwareController {

    private final S3FirmwareService s3Service;

    @PostMapping("/upload-url")
    public Map<String, String> getUploadUrl(@RequestParam String key) {
        String url = s3Service.generateUploadUrl(key);
        return Map.of(
                "uploadUrl", url,
                "key", key,
                "method", "PUT"
        );
    }

    @GetMapping("/download-url")
    public Map<String, String> getDownloadUrl(@RequestParam String key) {
        String url = s3Service.generateDownloadUrl(key);
        return Map.of(
                "downloadUrl", url,
                "key", key
        );
    }

    @DeleteMapping
    public Map<String, String> deleteFirmware(@RequestParam String key) {
        s3Service.deleteFile(key);
        return Map.of("status", "deleted", "key", key);
    }
}
