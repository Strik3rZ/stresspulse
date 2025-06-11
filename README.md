# StressPulse

Простой генератор нагрузки на CPU, память и сеть. Написал потому что надоело возиться с примитивными утилитами типа `stress` - хотелось что-то поумнее и с нормальным мониторингом.

Главная фишка - это "живая" нагрузка. Программа не просто жрёт процессор на одном уровне, а создаёт реалистичные паттерны с колебаниями, как в настоящих приложениях.

## Зачем это нужно

Иногда хочется протестировать как система ведёт себя под нагрузкой. Обычные стресс-тесты создают постоянную нагрузку, а в реальности она всегда скачет. Поэтому сделал инструмент, который умеет:

- Плавно менять нагрузку по разным паттернам
- Собирать метрики для Prometheus/Grafana  
- Работать в Docker и Kubernetes
- Не сожрать весь сервер случайно
- **Генерировать реалистичные логи как у настоящих приложений**
- **Создавать нагрузку на память с разными паттернами использования**
- **Тестировать RPS и время ответа веб-сервисов**

## Как попробовать

Самый быстрый способ:

```bash
# Скачай зависимости
go mod tidy

# Запусти на 50% CPU
go run main.go

# С фейковыми логами Java приложения
go run main.go -cpu 70 -fake-logs -fake-logs-type java

# Нагрузка на память 200MB постоянным паттерном
go run main.go -cpu 30 -memory -memory-target 200 -memory-pattern constant

# HTTP нагрузочное тестирование
go run main.go -http -http-url "http://localhost:8080/api/health" -http-rps 50

# Полный стресс-тест: CPU + память + HTTP + логи
go run main.go -cpu 60 -memory -memory-target 300 -http -http-url "http://localhost:8080" -http-rps 100 -fake-logs
```

Если есть Docker:

```bash
# Собери и запусти
docker build -t stresspulse .
docker run --rm stresspulse -cpu 70 -metrics -fake-logs -memory -memory-target 150
```

Для продакшна лучше взять готовый Docker Compose или Kubernetes:

```bash
# Всё окружение сразу (StressPulse + Prometheus + Grafana)
docker-compose up -d

# Или в Kubernetes  
./deploy.sh
```

Подробнее про развертывание читай в [DEPLOYMENT.md](DEPLOYMENT.md).

## Настройки

### Основные параметры
- `-cpu 50` - сколько процентов нагружать (0-100)
- `-duration 10m` - сколько работать (0 = бесконечно)  
- `-drift 20` - на сколько процентов может отклоняться от цели
- `-pattern sine` - какой паттерн использовать
- `-period 30s` - период одного цикла

### Нагрузка памяти
- `-memory` - включить нагрузку на память
- `-memory-target 200` - сколько MB памяти выделять
- `-memory-pattern constant` - паттерн использования: constant, leak, spike, cycle, random
- `-memory-interval 2s` - как часто менять состояние памяти

### HTTP нагрузочное тестирование
- `-http` - включить HTTP нагрузочное тестирование
- `-http-url "http://localhost:8080/api"` - URL для тестирования
- `-http-rps 100` - сколько запросов в секунду отправлять
- `-http-pattern constant` - паттерн RPS: constant, spike, cycle, ramp, random
- `-http-method POST` - HTTP метод (GET, POST, PUT, DELETE, PATCH)
- `-http-timeout 5s` - таймаут запросов
- `-http-headers "Content-Type:application/json,Authorization:Bearer token"` - заголовки
- `-http-body '{"test": "data"}'` - тело запроса

### Фейковые логи
- `-fake-logs` - включить генерацию фейковых логов  
- `-fake-logs-type java` - тип логов: java, web, microservice, database, ecommerce
- `-fake-logs-interval 1s` - как часто генерировать логи

### Остальное
- `-workers 4` - сколько потоков запустить (0 = по количеству ядер)
- `-log-level debug` - насколько подробные логи хочешь видеть
- `-metrics` - включить метрики для Prometheus
- `-save-profile` - сохранить результаты в JSON

## Примеры для жизни

```bash
# Лёгкий фоновый тест
go run main.go -cpu 30 -duration 30m

# Имитация пиковой нагрузки  
go run main.go -cpu 85 -pattern square -drift 15

# Тестирование REST API
go run main.go -http -http-url "http://localhost:3000/api/users" -http-rps 200 -http-method GET

# POST запросы с JSON
go run main.go -http -http-url "http://localhost:3000/api/users" -http-rps 50 -http-method POST \
  -http-headers "Content-Type:application/json" -http-body '{"name":"test","email":"test@example.com"}'

# Тестирование утечки памяти
go run main.go -cpu 40 -memory -memory-target 500 -memory-pattern leak -duration 1h

# Нагрузочное тестирование с постепенным увеличением RPS
go run main.go -http -http-url "http://localhost:8080" -http-rps 100 -http-pattern ramp -duration 30m

# Полный стресс для веб-сервиса
go run main.go -cpu 70 -memory -memory-target 400 -http -http-url "http://localhost:8080/api" \
  -http-rps 150 -http-pattern spike -fake-logs -fake-logs-type web -duration 2h

# Отладка (если что-то не работает)
go run main.go -log-level debug -cpu 25
```

## Паттерны нагрузки

Сделал несколько разных:

**sine** - плавные волны, как в обычных веб-приложениях

**square** - резкие скачки вверх-вниз, хорошо находит проблемы с масштабированием

**sawtooth** - медленно растёт и резко падает, помогает поймать утечки памяти

**random** - хаотичные колебания, максимально похоже на реальную нагрузку

Можешь поэкспериментировать и посмотреть как твоя система реагирует на разные типы.

## Паттерны использования памяти

Добавил реалистичные сценарии работы с памятью:

**constant** - постоянное выделение до целевого размера, как у стабильных приложений

**leak** - имитация утечки памяти, постепенное увеличение с редким освобождением  

**spike** - резкие скачки потребления памяти в 2-3 раза выше нормы

**cycle** - циклическое выделение и освобождение с периодом 2 минуты

**random** - хаотичные выделения разного размера, как в реальных приложениях

Удобно для тестирования поведения системы при разных сценариях использования памяти.

## Паттерны HTTP нагрузки

Добавил разные сценарии RPS для реалистичного тестирования:

**constant** - постоянная нагрузка, базовый тест производительности

**spike** - резкие спайки в 3 раза выше обычного, тестирует пиковые нагрузки

**cycle** - циклические изменения каждые 30 секунд, имитирует суточные колебания

**ramp** - постепенное увеличение нагрузки, тестирует масштабируемость

**random** - случайные колебания от 10% до 150% от целевого RPS

Отлично подходит для тестирования API, микросервисов и веб-приложений.

## Типы фейковых логов

Добавил реалистичные логи для разных типов приложений:

**java** - Spring Boot приложение с Hibernate, логи классов, SQL запросы, GC

**web** - HTTP access логи как в nginx/apache, разные статусы, времена ответа

**microservice** - современная архитектура с trace ID, span ID, circuit breaker

**database** - PostgreSQL/MySQL логи с запросами, временем выполнения, блокировками

**ecommerce** - события интернет-магазина: логины, покупки, платежи

**generic** - обычные логи приложения с разными уровнями

Логи генерируются в реальном времени параллельно с нагрузкой CPU. Удобно для тестирования систем сбора логов типа ELK, Fluentd, Loki.

## Мониторинг

Если запустишь с `-metrics`, то на порту 9090 появятся метрики для Prometheus:

### CPU метрики:
- `cpu_usage_percent` - сколько сейчас процессор нагружен
- `cpu_target_percent` - сколько хотели нагрузить  
- `pattern_value` - текущее значение паттерна
- `workers_count` - сколько потоков работает

### Память метрики:
- `memory_allocated_mb` - сколько памяти выделено сейчас
- `memory_target_mb` - целевое количество памяти
- `memory_total_allocated_bytes` - общий объём выделенной памяти  
- `memory_total_released_bytes` - общий объём освобождённой памяти
- `memory_allocation_operations_total` - количество операций выделения
- `memory_system_*` - системная статистика памяти Go runtime

### HTTP метрики:
- `http_requests_total` - общее количество запросов
- `http_requests_success_total` - успешные запросы
- `http_requests_failed_total` - неудачные запросы
- `http_requests_per_second` - текущий RPS
- `http_target_rps` - целевой RPS
- `http_response_time_seconds` - гистограмма времён ответа
- `http_avg_response_time_seconds` - среднее время ответа
- `http_success_rate_percent` - процент успешных запросов

Удобно смотреть в Grafana, особенно если используешь Docker Compose - там уже всё настроено.

## Время можно писать по-человечески

- `30s` - полминуты
- `5m` - пять минут  
- `2h` - два часа
- `1h30m` - полтора часа
- `500ms` - полсекунды (для интервала логов)

## Docker варианты

```bash
# Просто запустить
docker run --rm -p 9090:9090 stresspulse -cpu 70 -metrics

# С фейковыми логами
docker run --rm stresspulse -cpu 50 -fake-logs -fake-logs-type web

# С нагрузкой на память
docker run --rm stresspulse -cpu 40 -memory -memory-target 200

# HTTP нагрузочное тестирование внешнего сервиса
docker run --rm stresspulse -http -http-url "https://httpbin.org/get" -http-rps 20

# Ограничить ресурсы (чтобы не убить хост)
docker run --cpus 0.5 --memory 512m stresspulse -cpu 50 -memory -memory-target 100

# Полное окружение с мониторингом
docker-compose up -d
```

## Kubernetes

Есть готовые конфиги в папке `helm/cpu-stress`. Можно деплоить одной командой:

```bash
# Автоматический деплой
./deploy.sh

# Или руками
helm install stresspulse ./helm/cpu-stress

# Поменять настройки
helm upgrade stresspulse ./helm/cpu-stress --set config.cpu=80

# Посмотреть что происходит
make k8s-status
make k8s-logs
```

Поддерживает автомасштабирование, мониторинг через ServiceMonitor и всё такое.

## Сборка

```bash
# Обычная
go build -o stresspulse

# Для продакшна (меньше размер)
go build -ldflags="-s -w" -o stresspulse

# Для другой ОС
GOOS=linux GOARCH=amd64 go build -o stresspulse-linux
```

## Makefile

Сделал кучу удобных команд:

```bash
make help           # покажет все команды
make build          # собрать локально
make docker-build   # собрать Docker образ
make deploy         # задеплоить в Kubernetes
make clean          # убрать мусор
```

## Полезные команды

```bash
# Посмотреть нагрузку в реальном времени
htop

# Проверить метрики
curl http://localhost:9090/metrics

# Убить программу - она корректно завершится по Ctrl+C
```

## Структура проекта

Если хочешь покопаться в коде:

- `main.go` - точка входа, парсинг параметров
- `config/` - конфигурация и проверки
- `load/` - основная логика генерации нагрузки
- `patterns/` - алгоритмы для разных паттернов
- `logger/` - логирование с уровнями
- `logs/` - генератор фейковых логов
- `memory/` - генератор нагрузки памяти
- `network/` - HTTP нагрузочное тестирование
- `metrics/` - интеграция с Prometheus
- `helm/` - конфиги для Kubernetes
- `monitoring/` - настройки Prometheus/Grafana

## Что дальше

Планирую добавить:
- Нагрузку на диск (I/O операции)
- Больше паттернов 
- Web-интерфейс для управления
- Distributed режим для кластерного тестирования
- Больше типов фейковых логов (Kubernetes, Redis, ElasticSearch)
- WebSocket и gRPC нагрузочное тестирование
