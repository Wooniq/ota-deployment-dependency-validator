package com.ota.control.controller;

import com.ota.control.service.S3FirmwareService;
import lombok.RequiredArgsConstructor;
import org.springframework.web.bind.annotation.*;

import java.util.Map;

/**
 * 펌웨어 파일 관리 API
 * 과제 1번: S3 버킷에 파일을 업로드/다운로드 하는 애플리케이션
 */
@RestController
@RequestMapping("/api/firmware")
@RequiredArgsConstructor
@CrossOrigin(origins = "*")
public class FirmwareController {

    private final S3FirmwareService s3Service;

    /**
     * 펌웨어 업로드용 Presigned URL 발급
     * POST /api/firmware/upload-url?key=firmware/ADAS/v2.3.0.bin
     */
    @PostMapping("/upload-url")
    public Map<String, String> getUploadUrl(@RequestParam String key) {
        String url = s3Service.generateUploadUrl(key);
        return Map.of(
                "uploadUrl", url,
                "key", key,
                "method", "PUT"
        );
    }

    /**
     * 펌웨어 다운로드용 Presigned URL 발급
     * GET /api/firmware/download-url?key=firmware/ADAS/v2.3.0.bin
     */
    @GetMapping("/download-url")
    public Map<String, String> getDownloadUrl(@RequestParam String key) {
        String url = s3Service.generateDownloadUrl(key);
        return Map.of(
                "downloadUrl", url,
                "key", key
        );
    }

    /**
     * 펌웨어 파일 삭제
     * DELETE /api/firmware?key=firmware/ADAS/v2.3.0.bin
     */
    @DeleteMapping
    public Map<String, String> deleteFirmware(@RequestParam String key) {
        s3Service.deleteFile(key);
        return Map.of("status", "deleted", "key", key);
    }
}
