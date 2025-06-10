# Как запустить StressPulse

## Способы запуска

### Docker - самый простой способ

Если у тебя есть Docker, то это самый быстрый вариант:

```bash
# Сначала собираем образ
docker build -t stresspulse:latest .

# Запускаем контейнер
docker run --rm -p 9090:9090 stresspulse:latest
```

Всё! Приложение будет доступно на порту 9090.

### Docker Compose - для тех, кто хочет полный стек

Этот вариант я рекомендую для разработки. В одной команде поднимается всё сразу: само приложение, Prometheus для сбора метрик и Grafana для красивых графиков.

```bash
# Запускаем всё окружение
docker-compose up -d

# Смотрим логи (если что-то пошло не так)
docker-compose logs -f

# Когда надоело - выключаем
docker-compose down
```

После запуска у тебя будет:
- http://localhost:9090 - StressPulse с метриками
- http://localhost:9091 - Prometheus 
- http://localhost:3000 - Grafana (логин/пароль: admin/admin)

### Kubernetes - для продакшна

Тут уже посложнее, нужен кластер Kubernetes и Helm. Но я сделал скрипты, чтобы не мучиться с командами.

**Для Windows:**
```powershell
.\deploy.ps1
```

**Для Linux/Mac:**
```bash
./deploy.sh
```

Если хочешь всё сделать руками:

```bash
# Собираем образ
docker build -t stresspulse:latest .

# Деплоим через Helm
helm upgrade --install stresspulse ./helm/stresspulse \
    --namespace default \
    --set image.tag=latest \
    --wait

# Проверяем что всё работает
kubectl get pods -l app=stresspulse
```

Чтобы посмотреть метрики:
```bash
kubectl port-forward svc/stresspulse 9090:9090
```

Когда захочешь удалить:
```bash
helm uninstall stresspulse
```

## Makefile - для ленивых

Сделал кучу удобных команд, чтобы не запоминать длинные команды:

```bash
# Показать все команды
make help

# Собрать и запустить локально
make build
make run

# Docker штуки
make docker-build
make docker-run

# Docker Compose
make docker-compose-up
make docker-compose-down

# Kubernetes
make deploy
make k8s-status
make k8s-logs
```

## Настройки

Основные параметры можно поменять в `helm/cpu-stress/values.yaml`:

```yaml
config:
  cpu: 50          # Сколько процентов CPU нагружать
  drift: 20        # На сколько можно отклоняться от цели
  pattern: sine    # Какой паттерн использовать (sine, square, triangle)
  period: 30s      # Период одного цикла
  workers: 0       # Сколько потоков (0 = автоматически)
  logLevel: info   # Уровень логирования
```

Ресурсы (чтобы не сожрать весь сервер):
```yaml
resources:
  limits:
    cpu: 1000m
    memory: 128Mi
```

Если нужно автомасштабирование:
```yaml
autoscaling:
  enabled: true
  minReplicas: 1
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
```

## Что можно мониторить

Приложение выдаёт метрики для Prometheus:
- `cpu_usage_percent` - сколько сейчас нагружен CPU
- `cpu_target_percent` - сколько хотели нагрузить
- `pattern_value` - текущее значение паттерна
- `workers_count` - сколько потоков работает

Если используешь Docker Compose, то Grafana сама подхватит метрики и покажет графики.

## Когда что-то ломается

### Образ не собирается
Проверь что Docker запущен:
```bash
docker version
```

### Kubernetes не работает
Убедись что подключен к кластеру:
```bash
kubectl cluster-info
```

### Helm ругается
Возможно не установлен. На Linux/Mac:
```bash
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
```

### Нужно посмотреть логи
```bash
# Логи в Kubernetes
kubectl logs -f -l app=stresspulse

# Что происходит с подом
kubectl describe pod -l app=stresspulse

# События в кластере
kubectl get events --sort-by=.metadata.creationTimestamp
```