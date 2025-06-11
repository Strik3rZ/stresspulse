# Как запустить StressPulse

Тут я расскажу, как запустить этот генератор нагрузки. Есть несколько способов - выбирай какой удобнее.

## Способы запуска

### Docker - самый простой способ

Если у тебя есть Docker, то это самый быстрый вариант:

```bash
# Сначала собираем образ
docker build -t stresspulse:latest .

# Запускаем контейнер
docker run --rm -p 9090:9090 stresspulse:latest

# С нагрузкой на память
docker run --rm -p 9090:9090 --memory 512m stresspulse:latest -cpu 50 -memory -memory-target 200
```

Всё! Приложение будет доступно на порту 9090.

### Docker Compose - для тех, кто хочет полный стек

Этот вариант я рекомендую для разработки. В одной команде поднимается всё сразу: само приложение, Prometheus для сбора метрик и Grafana для красивых графиков.

```bash
# Запускаем всё окружение (включает CPU + память + логи)
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

В compose файле уже настроена нагрузка:
- 50% CPU с циклическим паттерном 
- 200MB памяти с циклическим паттерном
- Веб-логи каждую секунду

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

# Деплоим через Helm с нагрузкой памяти
helm upgrade --install stresspulse ./helm/cpu-stress \
    --namespace default \
    --set image.tag=latest \
    --set config.memoryEnabled=true \
    --set config.memoryTargetMB=300 \
    --set config.memoryPattern=spike \
    --wait

# Проверяем что всё работает
kubectl get pods -l app=stresspulse
```

Чтобы посмотреть метрики:
```bash
kubectl port-forward svc/stresspulse 9090:9090
# Откройте http://localhost:9090/metrics
```

Когда захочешь удалить:
```bash
helm uninstall stresspulse
```

## Makefile - для ленивых (как я)

Сделал кучу удобных команд, чтобы не запоминать длинные команды:

```bash
# Показать все команды
make help

# Собрать и запустить локально
make build
make run

# Docker
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

### CPU параметры:
```yaml
config:
  cpu: 50          # Сколько процентов CPU нагружать
  drift: 20        # На сколько можно отклоняться от цели
  pattern: sine    # Какой паттерн использовать (sine, square, triangle)
  period: 30s      # Период одного цикла
  workers: 0       # Сколько потоков (0 = автоматически)
  logLevel: info   # Уровень логирования
```

### Память параметры:
```yaml
config:
  memoryEnabled: true         # Включить нагрузку памяти
  memoryTargetMB: 200        # Целевое количество памяти в MB
  memoryPattern: cycle       # Паттерн: constant, leak, spike, cycle, random
  memoryInterval: 2s         # Интервал операций с памятью
```

### Фейковые логи:
```yaml
config:
  fakeLogsEnabled: true      # Включить генерацию логов
  fakeLogsType: web         # Тип: java, web, microservice, database, ecommerce
  fakeLogsInterval: 1s      # Частота генерации
```

Ресурсы (чтобы не сожрать весь сервер):
```yaml
resources:
  limits:
    cpu: 1000m     # Максимум 1 ядро CPU
    memory: 256Mi  # Максимум 256MB памяти (больше чем target для безопасности)
```

Если нужно автомасштабирование:
```yaml
autoscaling:
  enabled: true
  minReplicas: 1
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80
```

## Что можно мониторить

Приложение выдаёт метрики для Prometheus:

### CPU метрики:
- `cpu_usage_percent` - сколько сейчас нагружен CPU
- `cpu_target_percent` - сколько хотели нагрузить
- `pattern_value` - текущее значение паттерна
- `workers_count` - сколько потоков работает

### Память метрики:
- `memory_allocated_mb` - сколько памяти выделено приложением
- `memory_target_mb` - целевое количество памяти
- `memory_total_allocated_bytes` - общий объём выделенной памяти
- `memory_total_released_bytes` - общий объём освобождённой памяти
- `memory_system_*` - системная статистика Go runtime

Если используешь Docker Compose, то Grafana сама подхватит метрики и покажет графики.

## Примеры для разных сценариев

### Тестирование утечки памяти:
```bash
# Docker
docker run --rm --memory 512m stresspulse:latest \
  -cpu 30 -memory -memory-target 300 -memory-pattern leak -duration 1h

# Kubernetes
helm upgrade stresspulse ./helm/cpu-stress \
  --set config.cpu=30 \
  --set config.memoryEnabled=true \
  --set config.memoryTargetMB=300 \
  --set config.memoryPattern=leak
```

### Стресс-тест микросервиса:
```bash
# Docker с полной нагрузкой
docker run --rm --cpus 1 --memory 512m stresspulse:latest \
  -cpu 70 -memory -memory-target 200 -memory-pattern spike \
  -fake-logs -fake-logs-type microservice -metrics

# Kubernetes
helm upgrade stresspulse ./helm/cpu-stress \
  --set config.cpu=70 \
  --set config.memoryEnabled=true \
  --set config.memoryTargetMB=200 \
  --set config.memoryPattern=spike \
  --set config.fakeLogsEnabled=true \
  --set config.fakeLogsType=microservice
```

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

### Память не выделяется
Проверь лимиты контейнера - память target должна быть меньше memory limit:
```bash
# В Docker
docker run --memory 512m stresspulse -memory -memory-target 400  # OK
docker run --memory 256m stresspulse -memory -memory-target 400  # Будут проблемы
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

## Безопасность

Сделал всё по best practices:
- Приложение не запускается от root
- Убрал все лишние права
- Файловая система только для чтения
- Никаких повышений привилегий
- Память ограничена лимитами контейнера