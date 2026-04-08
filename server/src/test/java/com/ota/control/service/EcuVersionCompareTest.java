package com.ota.control.service;

import com.ota.control.domain.Ecu;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * Python is_compatible(current, required) 로직 검증
 * 튜플 비교를 Java isCompatibleWith()로 정확히 포팅했는지 확인
 */
class EcuVersionCompareTest {

    @Test
    @DisplayName("Major가 큰 경우 → 호환")
    void majorGreater() {
        Ecu ecu = Ecu.builder().major(2).minor(0).patch(0).build();
        assertThat(ecu.isCompatibleWith(1, 5, 0)).isTrue();
    }

    @Test
    @DisplayName("Major 같고 Minor 큰 경우 → 호환")
    void minorGreater() {
        Ecu ecu = Ecu.builder().major(1).minor(5).patch(0).build();
        assertThat(ecu.isCompatibleWith(1, 2, 0)).isTrue();
    }

    @Test
    @DisplayName("정확히 같은 버전 → 호환")
    void exactMatch() {
        Ecu ecu = Ecu.builder().major(1).minor(2).patch(0).build();
        assertThat(ecu.isCompatibleWith(1, 2, 0)).isTrue();
    }

    @Test
    @DisplayName("Minor가 작은 경우 → 비호환")
    void minorLess() {
        Ecu ecu = Ecu.builder().major(1).minor(0).patch(0).build();
        assertThat(ecu.isCompatibleWith(1, 2, 0)).isFalse();
    }

    @Test
    @DisplayName("Patch만 작은 경우 → 비호환")
    void patchLess() {
        Ecu ecu = Ecu.builder().major(1).minor(2).patch(0).build();
        assertThat(ecu.isCompatibleWith(1, 2, 3)).isFalse();
    }
}
