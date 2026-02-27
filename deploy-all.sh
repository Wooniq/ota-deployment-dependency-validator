#!/bin/bash

# 1. 차량 데이터 생성 (Python 실행)
echo "🚀 Generating 1,000 vehicle data files..."
python3 cmd/agent/gen_vehicles.py

# 2. 기존 ConfigMap 삭제 (업데이트를 위해)
kubectl delete configmap ota-inventory --ignore-not-found

# 3. 생성된 데이터를 기반으로 ConfigMap 생성
# gen_vehicles.py가 생성하는 경로인 ./data/inventory 를 참조
echo "📦 Creating ConfigMap from generated data..."
kubectl create configmap ota-inventory --from-file=data/inventory/

# 4. StatefulSet 배포 (또는 재기동)
echo "⚓ Deploying ota-agent StatefulSet..."
kubectl apply -f ota-agent-ss.yaml

# 5. 변경사항 반영을 위한 롤아웃 (이미 배포 중일 경우)
kubectl rollout restart statefulset ota-agent

echo "✅ Deployment complete! Waiting for pods to be Running..."
