# 통합 배포 스크립트
#!/bin/bash
set -e # 에러 발생 시 즉시 중단

echo "🚀 [1/4] 데이터 생성: 1,000대 차량 인벤토리 굽는 중..."
python3 agent/gen_vehicles.py

echo "📦 [2/4] 설정 동기화: ConfigMap 업데이트..."
kubectl delete configmap ota-inventory --ignore-not-found
kubectl create configmap ota-inventory --from-file=cmd/data/inventory/

echo "⚓ [3/4] 서버 가동: OTA 관제 서버 배포..."
kubectl apply -f infra/ota-server-dep.yaml

echo "🚜 [4/4] 시뮬레이션 가동: 에이전트 1,000대 투입..."
kubectl apply -f infra/ota-agent-ss.yaml

# 롤아웃 상태 감시 (선택 사항)
# kubectl rollout status statefulset ota-agent --timeout=5m

echo "✅ 모든 시스템 배포 완료! 'kubectl get pods'로 확인하세요."

echo "🚚Distributing image to worker nodes..."
for node in worker1 worker2 worker3; do
  ssh admin@$node "docker load" < ota-agent-v4.tar &
done
wait
