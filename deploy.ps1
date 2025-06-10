param(
    [string]$ImageTag = "latest",
    [string]$Namespace = "default"
)

# Переменные
$ImageName = "stresspulse"
$ReleaseName = "stresspulse"

Write-Host "Начинаем deployment StressPulse..." -ForegroundColor Yellow

# Сборка Docker образа
Write-Host "Собираем Docker образ..." -ForegroundColor Yellow
docker build -t "${ImageName}:${ImageTag}" .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Docker образ успешно собран" -ForegroundColor Green
} else {
    Write-Host "Ошибка при сборке Docker образа" -ForegroundColor Red
    exit 1
}

# Проверка подключения к Kubernetes кластеру
Write-Host "🔍 Проверяем подключение к Kubernetes..." -ForegroundColor Yellow
kubectl cluster-info | Out-Null

if ($LASTEXITCODE -eq 0) {
    Write-Host "Подключение к Kubernetes кластеру установлено" -ForegroundColor Green
} else {
    Write-Host "Нет подключения к Kubernetes кластеру" -ForegroundColor Red
    exit 1
}

# Создание namespace, если не существует
kubectl create namespace $Namespace --dry-run=client -o yaml | kubectl apply -f -

# Развертывание с помощью Helm
Write-Host "Развертываем приложение с помощью Helm..." -ForegroundColor Yellow
helm upgrade --install $ReleaseName ./helm/cpu-stress --namespace $Namespace --set image.tag=$ImageTag --wait

if ($LASTEXITCODE -eq 0) {
    Write-Host "Deployment успешно завершен!" -ForegroundColor Green
    
    # Показываем статус
    Write-Host "Статус развертывания:" -ForegroundColor Yellow
    kubectl get pods -n $Namespace -l app=stresspulse
    Write-Host ""
    kubectl get svc -n $Namespace -l app=stresspulse
    
    # Показываем команды для мониторинга
    Write-Host "Полезные команды:" -ForegroundColor Yellow
    Write-Host "Просмотр логов: kubectl logs -f -n $Namespace -l app=stresspulse"
    Write-Host "Просмотр метрик: kubectl port-forward -n $Namespace svc/stresspulse 9090:9090"
    Write-Host "Удаление: helm uninstall $ReleaseName -n $Namespace"
    
} else {
    Write-Host "Ошибка при развертывании" -ForegroundColor Red
    exit 1
} 