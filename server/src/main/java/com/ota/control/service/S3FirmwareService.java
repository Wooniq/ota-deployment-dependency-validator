package com.ota.control.service;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.*;
import software.amazon.awssdk.services.s3.presigner.S3Presigner;
import software.amazon.awssdk.services.s3.presigner.model.GetObjectPresignRequest;
import software.amazon.awssdk.services.s3.presigner.model.PutObjectPresignRequest;

import java.time.Duration;

/**
 * S3 펌웨어 파일 관리 서비스
 * 과제 1번: S3 버킷에 파일 업로드/다운로드 애플리케이션
 */
@Service
@Slf4j
public class S3FirmwareService {

    private final S3Client s3Client;
    private final S3Presigner s3Presigner;

    @Value("${aws.s3.bucket-firmware}")
    private String firmwareBucket;

    public S3FirmwareService(S3Client s3Client, S3Presigner s3Presigner) {
        this.s3Client = s3Client;
        this.s3Presigner = s3Presigner;
    }

    /**
     * 펌웨어 업로드용 Presigned URL 발급 (유효시간 10분)
     */
    public String generateUploadUrl(String key) {
        PutObjectPresignRequest request = PutObjectPresignRequest.builder()
                .signatureDuration(Duration.ofMinutes(10))
                .putObjectRequest(PutObjectRequest.builder()
                        .bucket(firmwareBucket)
                        .key(key)
                        .build())
                .build();

        String url = s3Presigner.presignPutObject(request).url().toString();
        log.info("[S3] Upload URL 발급: key={}", key);
        return url;
    }

    /**
     * 펌웨어 다운로드용 Presigned URL 발급 (유효시간 30분)
     */
    public String generateDownloadUrl(String key) {
        GetObjectPresignRequest request = GetObjectPresignRequest.builder()
                .signatureDuration(Duration.ofMinutes(30))
                .getObjectRequest(GetObjectRequest.builder()
                        .bucket(firmwareBucket)
                        .key(key)
                        .build())
                .build();

        String url = s3Presigner.presignGetObject(request).url().toString();
        log.info("[S3] Download URL 발급: key={}", key);
        return url;
    }

    /**
     * S3 파일 삭제
     */
    public void deleteFile(String key) {
        s3Client.deleteObject(DeleteObjectRequest.builder()
                .bucket(firmwareBucket)
                .key(key)
                .build());
        log.info("[S3] 파일 삭제: key={}", key);
    }
}
