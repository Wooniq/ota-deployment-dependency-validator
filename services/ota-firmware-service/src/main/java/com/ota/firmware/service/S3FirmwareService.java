package com.ota.firmware.service;

import lombok.extern.slf4j.Slf4j;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Service;
import software.amazon.awssdk.services.s3.S3Client;
import software.amazon.awssdk.services.s3.model.DeleteObjectRequest;
import software.amazon.awssdk.services.s3.model.GetObjectRequest;
import software.amazon.awssdk.services.s3.model.PutObjectRequest;
import software.amazon.awssdk.services.s3.presigner.S3Presigner;
import software.amazon.awssdk.services.s3.presigner.model.GetObjectPresignRequest;
import software.amazon.awssdk.services.s3.presigner.model.PutObjectPresignRequest;

import java.time.Duration;

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

    public String generateUploadUrl(String key) {
        PutObjectPresignRequest request = PutObjectPresignRequest.builder()
                .signatureDuration(Duration.ofMinutes(10))
                .putObjectRequest(PutObjectRequest.builder()
                        .bucket(firmwareBucket)
                        .key(key)
                        .build())
                .build();

        String url = s3Presigner.presignPutObject(request).url().toString();
        log.info("[S3] Upload URL issued: key={}", key);
        return url;
    }

    public String generateDownloadUrl(String key) {
        GetObjectPresignRequest request = GetObjectPresignRequest.builder()
                .signatureDuration(Duration.ofMinutes(30))
                .getObjectRequest(GetObjectRequest.builder()
                        .bucket(firmwareBucket)
                        .key(key)
                        .build())
                .build();

        String url = s3Presigner.presignGetObject(request).url().toString();
        log.info("[S3] Download URL issued: key={}", key);
        return url;
    }

    public void deleteFile(String key) {
        s3Client.deleteObject(DeleteObjectRequest.builder()
                .bucket(firmwareBucket)
                .key(key)
                .build());
        log.info("[S3] File deleted: key={}", key);
    }
}
