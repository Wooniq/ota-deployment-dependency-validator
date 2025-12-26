from fastapi import FastAPI
from pydantic import BaseModel
from typing import List
from backend.validator import validate_ota

app = FastAPI(title="OTA 의존성 검증 시스템")

# --- 데이터 규격 정의 (입력값 체크용) ---
class ECUInfo(BaseModel):
    type: str
    major: int
    minor: int
    patch: int

class DependencyRule(BaseModel):
    required_type: str
    min_major: int
    min_minor: int
    min_patch: int

class CheckRequest(BaseModel):
    vehicle_id: str
    package_id: str
    ecus: List[ECUInfo]
    rules: List[DependencyRule]

# --- API 엔드포인트 ---
@app.post("/check-update")
async def check_update(request: CheckRequest):
    """
    차량 정보와 규칙을 JSON으로 받아 업데이트 가능 여부 반환
    """
    # 받은 데이터를 validator가 읽을 수 있는 리스트/딕셔너리 형태로 변환
    vehicle_ecus = [ecu.dict() for ecu in request.ecus]
    rules_data = [rule.dict() for rule in request.rules]

    # 검증 로직 실행
    success, report = validate_ota(vehicle_ecus, rules_data)

    return {
        "vehicle_id": request.vehicle_id,
        "package_id": request.package_id,
        "is_available": success,
        "details": report
    }

@app.get("/")
def read_root():
    return {"message": "OTA Validator API 가동 중 ..."}