from backend.validator import validate_ota

# --- 모듈 독립 테스트 (더미 데이터) ---
def run_test_scenario(name, ecus, rules):
    print(f"\n>>> 시나리오: {name}")
    success, report = validate_ota(ecus, rules)
    print(f"결과: {'배포 가능' if success else '배포 불가'}")
    for r in report:
        print(f" - {r['rule']}: {r['status']} ({r.get('reason', '정상')})")

if __name__ == "__main__":
    # 시나리오 1: 버전 미달 (실패)
    run_test_scenario(
        "BCM 버전 미달",
        [{"type": "BCM", "major": 1, "minor": 1, "patch": 5}],
        [{"required_type": "BCM", "min_major": 1, "min_minor": 2, "min_patch": 0}]
    )

    # 시나리오 2: 버전 일치 (성공)
    run_test_scenario(
        "BCM 버전 충족",
        [{"type": "BCM", "major": 1, "minor": 2, "patch": 0}],
        [{"required_type": "BCM", "min_major": 1, "min_minor": 2, "min_patch": 0}]
    )

    # 시나리오 3: 제어기 누락 (실패)
    run_test_scenario(
        "필수 제어기(VCU) 없음",
        [{"type": "BMS", "major": 2, "minor": 0, "patch": 0}],
        [{"required_type": "VCU", "min_major": 1, "min_minor": 0, "patch": 0}]
    )