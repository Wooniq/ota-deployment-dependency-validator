"""
OTA(Over-the-Air) 업데이트 의존성 검증 엔진
차량의 현재 SW 버전과 업데이트 패키지의 요구 사양을 비교하여 배포 가능 여부 판별
"""

def is_compatible(current, required):
    """
    [버전 호환성 판별 함수]
    Semantic Versioning 규칙에 따라 현재 버전이 최소 요구 사양 이상인지 체크

    Args:
        current (tuple): 현재 버전 (Major, Minor, Patch) - 예: (1, 5, 2)
        required (tuple): 최소 요구 버전 (Major, Minor, Patch) - 예: (1, 2, 0)

    Returns:
        bool: 호환 가능하면 True, 미달이면 False
    """
    # 파이썬의 튜플 비교는 첫 번째 요소(Major)부터 순차적으로 비교하므로
    # 별도의 복잡한 if문 없이도 완벽한 버전 비교 연산 가능
    return current >= required


def validate_ota(vehicle_ecus, rules):
    """
    [OTA 배포 사전 검증 메인 로직]
    차량 내 여러 제어기(ECU)의 상태를 조사하여 모든 의존성 규칙을 만족하는지 검사

    Args:
        vehicle_ecus (list): 차량에 장착된 ECU 정보 리스트
                             (예: [{'type': 'BMS', 'major': 2, ...}, ...])
        rules (list): 업데이트 패키지에 설정된 제어기별 최소 버전 규칙 리스트
                      (예: [{'required_type': 'BCM', 'min_major': 1, ...}, ...])

    Returns:
        tuple: (전체 통과 여부[bool], 상세 검증 결과 리포트[list])
    """
    # 각 규칙별 검증 결과를 담을 리스트
    validation_report = []

    for rule in rules:
        # 1. 대상 차량에서 규칙이 요구하는 제어기(ECU)가 존재하는지 탐색
        target_ecu = next((e for e in vehicle_ecus if e['type'] == rule['required_type']), None)

        # 2. 제어기가 누락된 경우 (심각한 예외 상황)
        if not target_ecu:
            validation_report.append({
                "rule": rule['required_type'],
                "status": "FAIL",
                "reason": "검증에 필요한 제어기를 차량에서 찾을 수 없습니다."
            })
            continue

        # 3. 버전 정보 추출 및 튜플화 (비교 연산 준비)
        current_v = (target_ecu['major'], target_ecu['minor'], target_ecu['patch'])
        required_v = (rule['min_major'], rule['min_minor'], rule['min_patch'])

        # 4. 버전 비교 수행
        if is_compatible(current_v, required_v):
            # 성공 시 결과 기록
            validation_report.append({
                "rule": rule['required_type'],
                "status": "PASS",
                "current_version": f"{current_v[0]}.{current_v[1]}.{current_v[2]}",
                "required_version": f"{required_v[0]}.{required_v[1]}.{required_v[2]}"
            })
        else:
            # 실패 시 사유 기록 (디버깅용)
            validation_report.append({
                "rule": rule['required_type'],
                "status": "FAIL",
                "reason": f"버전 미달 (현재: v{current_v[0]}.{current_v[1]}.{current_v[2]} < 요구: v{required_v[0]}.{required_v[1]}.{required_v[2]})"
            })

    # [최종 판결] 모든 검사항목의 status가 "PASS"인 경우에만 전체 True 반환
    is_all_passed = all(item['status'] == "PASS" for item in validation_report)

    return is_all_passed, validation_report