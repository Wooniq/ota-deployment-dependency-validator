import json
import os
import random
import string

# 1. 현재 스크립트(agent/gen_vehicles.py)의 절대 경로 추출
current_script_path = os.path.abspath(__file__)
# 2. 스크립트가 속한 폴더(agent/) 추출
current_dir = os.path.dirname(current_script_path)
# 3. 한 단계 위인 루트 폴더 아래의 data/inventory 경로 설정
DATA_DIR = os.path.join(current_dir, "..", "data", "inventory")

# 경로 정규화 (../ 등을 깔끔하게 정리)
DATA_DIR = os.path.normpath(DATA_DIR)

# 폴더 생성
os.makedirs(DATA_DIR, exist_ok=True)

def generate_vin(index):
    """ISO 3779 표준 준수: 17자리, 금지문자(I, O, Q) 제외"""
    allowed_chars = "".join([c for c in string.ascii_uppercase + string.digits if c not in 'IOQ'])
    wmi = "WNK"  # World Manufacturer Identifier
    vds = "SDV77" # Vehicle Descriptor Section
    vis = f"{index:06d}" # Vehicle Identifier Section (일련번호)

    # 9번째 체크 디지트 'X'와 연식 코드 'R' 포함
    vin = f"{wmi}{vds}XR{vis}"

    # 부족한 자릿수를 허용된 문자로 채워 17자리 완성
    while len(vin) < 17:
        vin = vin[:9] + random.choice(allowed_chars) + vin[9:]

    return vin[:17]

def generate_vehicles(count=1000):
    # DLT APID(4자 고정)를 위한 명시적 패딩 적용
    # 4바이트 패딩을 데이터 소스 레벨에서 확정 지음으로, Go 에이전트의 런타임 오버헤드를 줄일 수 있음
    ecu_pool = [
        {"id": "BMS ", "hw": "HW-B1"},
        {"id": "ICU ", "hw": "HW-I2"},
        {"id": "VCU ", "hw": "HW-V3"},
        {"id": "ADAS", "hw": "HW-A1"},
        {"id": "TCU ", "hw": "HW-T1"}
    ]

    for i in range(1, count + 1):
        file_name = f"vehicle_{i:04d}.json"

        # 차량마다 무작위로 2~4개의 ECU 장착
        selected_ecus = random.sample(ecu_pool, random.randint(2, 4))

        ecu_info_list = []
        for ecu in selected_ecus:
            # SemVer 2.0.0 (Major.Minor.Patch) + SAP HANA 일관성 (v 접두사)
            sw_version = f"v{random.randint(1, 2)}.{random.randint(0, 5)}.{random.randint(0, 10)}"
            ecu_info_list.append({
                "id": ecu["id"],
                "hw_version": ecu["hw"],
                "sw_version": sw_version
            })

        # soh 필드 추가
        # 업데이트 가능 상태(0.3 이상)를 보장하기 위해 0.85~0.99 사이의 값 생성
        soh_value = round(random.uniform(0.85, 0.99), 2)

        vehicle_data = {
            "vin": generate_vin(i),
            "soh": soh_value,
            "ecus": ecu_info_list
        }

        with open(os.path.join(DATA_DIR, file_name), 'w') as f:
            json.dump(vehicle_data, f, indent=4)

    print(f"{count}개의 표준 규격 차량 데이터가 {DATA_DIR}에 생성됨")

if __name__ == "__main__":
    generate_vehicles()
