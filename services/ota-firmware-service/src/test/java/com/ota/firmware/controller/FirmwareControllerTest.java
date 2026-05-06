package com.ota.firmware.controller;

import com.ota.firmware.service.S3FirmwareService;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.WebMvcTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.test.web.servlet.MockMvc;

import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.delete;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@WebMvcTest({FirmwareController.class, HealthController.class})
class FirmwareControllerTest {

    @Autowired
    private MockMvc mockMvc;

    @MockBean
    private S3FirmwareService s3FirmwareService;

    @Test
    @DisplayName("업로드 Presigned URL 응답을 반환")
    void shouldReturnUploadUrl() throws Exception {
        when(s3FirmwareService.generateUploadUrl("firmware/BMS/v3.0.0.bin"))
                .thenReturn("https://s3.example/upload");

        mockMvc.perform(post("/api/firmware/upload-url")
                        .param("key", "firmware/BMS/v3.0.0.bin"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.uploadUrl").value("https://s3.example/upload"))
                .andExpect(jsonPath("$.key").value("firmware/BMS/v3.0.0.bin"))
                .andExpect(jsonPath("$.method").value("PUT"));
    }

    @Test
    @DisplayName("다운로드 Presigned URL 응답을 반환")
    void shouldReturnDownloadUrl() throws Exception {
        when(s3FirmwareService.generateDownloadUrl("firmware/BMS/v3.0.0.bin"))
                .thenReturn("https://s3.example/download");

        mockMvc.perform(get("/api/firmware/download-url")
                        .param("key", "firmware/BMS/v3.0.0.bin"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.downloadUrl").value("https://s3.example/download"))
                .andExpect(jsonPath("$.key").value("firmware/BMS/v3.0.0.bin"));
    }

    @Test
    @DisplayName("펌웨어 삭제 요청을 서비스에 위임")
    void shouldDeleteFirmware() throws Exception {
        mockMvc.perform(delete("/api/firmware")
                        .param("key", "firmware/BMS/v3.0.0.bin"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("deleted"))
                .andExpect(jsonPath("$.key").value("firmware/BMS/v3.0.0.bin"));

        verify(s3FirmwareService).deleteFile("firmware/BMS/v3.0.0.bin");
    }
}
