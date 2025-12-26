import os
from dotenv import load_dotenv

load_dotenv()

def get_connection():
    try:
        # DB 주소가 없을 경우를 위한 예외 처리
        address = os.getenv("HANA_ADDRESS")
        if not address:
            return None

        from hdbcli import dbapi
        conn = dbapi.connect(
            address=address,
            port=int(os.getenv("HANA_PORT", 30015)),
            user=os.getenv("HANA_USER"),
            password=os.getenv("HANA_PWD")
        )
        return conn
    except Exception as e:
        print(f"DB 연결 실패: {e}")
        return None