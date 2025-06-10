#!/bin/bash

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

IMAGE_NAME="stresspulse"
IMAGE_TAG=${1:-latest}
NAMESPACE=${2:-default}
RELEASE_NAME="stresspulse"

echo -e "${YELLOW} Начинаем deployment StressPulse...${NC}"

echo -e "${YELLOW} Собираем Docker образ...${NC}"
docker build -t ${IMAGE_NAME}:${IMAGE_TAG} .

if [ $? -eq 0 ]; then
    echo -e "${GREEN} Docker образ успешно собран${NC}"
else
    echo -e "${RED} Ошибка при сборке Docker образа${NC}"
    exit 1
fi

echo -e "${YELLOW} Проверяем подключение к Kubernetes...${NC}"
kubectl cluster-info > /dev/null 2>&1

if [ $? -eq 0 ]; then
    echo -e "${GREEN} Подключение к Kubernetes кластеру установлено${NC}"
else
    echo -e "${RED} Нет подключения к Kubernetes кластеру${NC}"
    exit 1
fi

kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -

echo -e "${YELLOW} Развертываем приложение с помощью Helm...${NC}"
helm upgrade --install ${RELEASE_NAME} ./helm/cpu-stress \
    --namespace ${NAMESPACE} \
    --set image.tag=${IMAGE_TAG} \
    --wait

if [ $? -eq 0 ]; then
    echo -e "${GREEN} Deployment успешно завершен!${NC}"
    
    echo -e "${YELLOW} Статус развертывания:${NC}"
    kubectl get pods -n ${NAMESPACE} -l app=stresspulse
    echo ""
    kubectl get svc -n ${NAMESPACE} -l app=stresspulse
    
    echo -e "${YELLOW} Полезные команды:${NC}"
    echo "Просмотр логов: kubectl logs -f -n ${NAMESPACE} -l app=stresspulse"
    echo "Просмотр метрик: kubectl port-forward -n ${NAMESPACE} svc/stresspulse 9090:9090"
    echo "Удаление: helm uninstall ${RELEASE_NAME} -n ${NAMESPACE}"
    
else
    echo -e "${RED} Ошибка при развертывании${NC}"
    exit 1
fi 