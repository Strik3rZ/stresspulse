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
- **WebSocket нагрузочное тестирование**
- **gRPC тестирование**
- **Управлять всем через современный веб-интерфейс**
- **Распределённое тестирование с несколькими агентами**

## Веб-интерфейс управления

Теперь можно управлять StressPulse через современный веб-интерфейс:

```bash
# Запуск с веб-интерфейсом
go run main.go -web

# На другом порту
go run main.go -web -web-port 9000
```

**Возможности веб-интерфейса:**
- **Управление в реальном времени** - запуск/остановка/настройка без перезапуска
- **Live мониторинг** - метрики CPU, памяти, HTTP, WebSocket, gRPC в реальном времени
- **Гибкие настройки** - все параметры доступны через удобные формы
- **Логи в браузере** - просмотр событий с цветовой кодировкой
- **Автосохранение** - конфигурация сохраняется в браузере
- **Responsive** - работает на десктопе, планшете и мобильном
- **Управление агентами** - добавление, настройка и мониторинг удалённых агентов

После запуска с флагом `-web` откройте браузер: **http://localhost:8080**

Подробнее в [WEB_INTERFACE_GUIDE.md](WEB_INTERFACE_GUIDE.md).

## Распределённое тестирование с агентами

Одна из самых крутых фишек - возможность запускать нагрузочные тесты на нескольких машинах одновременно. Это особенно полезно когда нужно создать реально высокую нагрузку или протестировать систему с разных точек сети.

### Как это работает

**Мастер-сервер** (с веб-интерфейсом) управляет **агентами** на других машинах. Через веб-интерфейс можно:
- Добавлять агентов по IP/URL
- Настраивать разные типы нагрузки для каждого агента
- Запускать/останавливать тесты на всех агентах сразу
- Мониторить статистику в реальном времени
- Агенты автоматически сохраняются и восстанавливаются после перезапуска

### Быстрый старт с агентами

```bash
# 1. Запускаем мастер-сервер с веб-интерфейсом
./stresspulse -web -web-port 8080

# 2. На других машинах запускаем агентов
./stresspulse -agent -agent-port 8081
./stresspulse -agent -agent-port 8082  # если на той же машине

# 3. Открываем http://localhost:8080 в браузере
# 4. В разделе "Agent Management" добавляем агентов:
#    - Agent ID: server1, URL: http://192.168.1.100:8081
#    - Agent ID: server2, URL: http://192.168.1.101:8081

# 5. Настраиваем нагрузку для каждого агента отдельно
# 6. Запускаем тесты одной кнопкой!
```

### Возможности агентов

**Индивидуальная настройка** - каждый агент может выполнять свой тип нагрузки:
- Агент 1: CPU нагрузка 70% + генерация логов
- Агент 2: HTTP тестирование API с 500 RPS  
- Агент 3: WebSocket соединения + память 1GB
- Агент 4: gRPC тестирование микросервисов

**Удобное управление через веб-интерфейс:**
- CPU - настройка процентов, паттернов, дрифта
- Memory - выбор размера и паттерна использования  
- HTTP - URL, RPS, методы, заголовки
- WebSocket - подключения, сообщения, интервалы
- gRPC - адреса, RPS, методы, безопасность

**Мониторинг в реальном времени:**
- Статус каждого агента (онлайн/офлайн)
- Детальная статистика по каждому типу нагрузки
- Время последнего подключения
- Системная информация (CPU, память, горутины)

**Автоматическое сохранение:**
- Агенты сохраняются в `config/agents.json`
- После перезапуска мастера все агенты восстанавливаются
- Автоматические health check'и каждые 30 секунд

### Примеры использования агентов

```bash
# Тестирование веб-приложения с разных точек
# Агент 1 (Москва): HTTP нагрузка на API
# Агент 2 (СПб): WebSocket подключения к чату  
# Агент 3 (Екб): gRPC вызовы к микросервисам

# Нагрузочное тестирование базы данных
# Агент 1: SELECT запросы через HTTP API
# Агент 2: INSERT/UPDATE через другой API
# Агент 3: Фоновая нагрузка на CPU для имитации других процессов

# Тестирование CDN и балансировщиков
# Агенты в разных регионах делают запросы к одному URL
# Можно увидеть разницу в производительности по регионам
```

### Команды для агентов

```bash
# Запуск агента на стандартном порту
./stresspulse -agent

# Агент на другом порту  
./stresspulse -agent -agent-port 8082

# Агент с отладочными логами
./stresspulse -agent -log-level debug

# Проверить что агент работает
curl http://localhost:8081/health
```

**Агенты автоматически:**
- Отвечают на health check'и
- Принимают команды запуска/остановки тестов
- Отправляют статистику обратно мастеру
- Логируют все операции для отладки
- Корректно завершаются по Ctrl+C

Это делает StressPulse идеальным для серьёзного нагрузочного тестирования, когда нужна реальная распределённая нагрузка!

## Как попробовать

Самый быстрый способ:

```bash
# Скачай зависимости
go mod tidy

# Запусти на 50% CPU
go run main.go

# С веб-интерфейсом (откроется на http://localhost:8080)
go run main.go -web

# С фейковыми логами Java приложения
go run main.go -cpu 70 -fake-logs -fake-logs-type java

# Нагрузка на память 200MB постоянным паттерном
go run main.go -cpu 30 -memory -memory-target 200 -memory-pattern constant

# HTTP нагрузочное тестирование
go run main.go -http -http-url "http://localhost:8080/api/health" -http-rps 50

# WebSocket тест
go run main.go -websocket -websocket-url "ws://echo.websocket.org" -websocket-cps 10

# gRPC тест
go run main.go -grpc -grpc-addr "localhost:9000" -grpc-rps 20

# Полный стресс-тест: CPU + память + HTTP + WebSocket + gRPC + логи + веб-интерфейс
go run main.go -cpu 60 -memory -memory-target 300 -http -http-url "http://localhost:8080" -http-rps 100 -websocket -websocket-url "ws://localhost:8080/ws" -grpc -grpc-addr "localhost:9000" -fake-logs -web

**Распределённое тестирование с агентами:**

```bash
# Запуск мастер-сервера с веб-интерфейсом
go run main.go -web -web-port 8080

# На других машинах/терминалах запускаем агентов
go run main.go -agent -agent-port 8081
go run main.go -agent -agent-port 8082

# Теперь в браузере http://localhost:8080 можно:
# - Добавить агентов через Agent Management
# - Настроить разную нагрузку для каждого агента
# - Запустить все тесты одновременно
# - Мониторить статистику в реальном времени

# Пример: один агент делает HTTP нагрузку, другой - CPU + память
# Агент 1: HTTP тестирование API
# Агент 2: CPU 70% + Memory 500MB + фейковые логи
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

### WebSocket тестирование
- `-websocket` - включить WebSocket нагрузочное тестирование
- `-websocket-url "ws://localhost:8080/ws"` - WebSocket URL  
- `-websocket-cps 10` - сколько новых соединений в секунду создавать
- `-websocket-pattern constant` - паттерн подключений
- `-websocket-message-interval 1s` - как часто отправлять сообщения
- `-websocket-message-size 256` - размер сообщений в байтах
- `-websocket-headers "Origin:example.com"` - заголовки подключения

### gRPC тестирование
- `-grpc` - включить gRPC нагрузочное тестирование
- `-grpc-addr "localhost:9000"` - адрес gRPC сервера
- `-grpc-rps 50` - сколько запросов в секунду
- `-grpc-pattern constant` - паттерн нагрузки
- `-grpc-method health_check` - тип метода (health_check, unary, server_stream, client_stream, bidi_stream)
- `-grpc-service "UserService"` - имя сервиса для health check
- `-grpc-secure` - использовать TLS
- `-grpc-metadata "auth:token,version:v1"` - метаданные

### Фейковые логи
- `-fake-logs` - включить генерацию фейковых логов  
- `-fake-logs-type java` - тип логов: java, web, microservice, database, ecommerce
- `-fake-logs-interval 1s` - как часто генерировать логи

### Веб-интерфейс и агенты
- `-web` - запустить веб-интерфейс для управления
- `-web-port 8080` - порт для веб-интерфейса (по умолчанию 8080)
- `-agent` - запустить в режиме агента (принимает команды от мастера)
- `-agent-port 8081` - порт для агента (по умолчанию 8081)

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

# WebSocket чат симуляция
go run main.go -websocket -websocket-url "ws://localhost:8080/chat" -websocket-cps 20 \
  -websocket-message-interval 2s -websocket-message-size 128

# gRPC микросервис тест
go run main.go -grpc -grpc-addr "user-service:9000" -grpc-rps 100 -grpc-secure \
  -grpc-service "UserService" -grpc-metadata "tenant:prod"

# Тестирование утечки памяти
go run main.go -cpu 40 -memory -memory-target 500 -memory-pattern leak -duration 1h

# Нагрузочное тестирование с постепенным увеличением RPS
go run main.go -http -http-url "http://localhost:8080" -http-rps 100 -http-pattern ramp -duration 30m

# Полный стресс для веб-сервиса
go run main.go -cpu 70 -memory -memory-target 400 \
  -http -http-url "http://localhost:8080/api" -http-rps 150 -http-pattern spike \
  -websocket -websocket-url "ws://localhost:8080/ws" -websocket-cps 25 \
  -grpc -grpc-addr "localhost:9000" -grpc-rps 75 \
  -fake-logs -fake-logs-type web -duration 2h

# Отладка
go run main.go -log-level debug -cpu 25
```

**Распределённое тестирование:**

```bash
# Мастер-сервер с веб-интерфейсом
go run main.go -web -web-port 8080

# Агенты на разных машинах
go run main.go -agent -agent-port 8081  # Машина 1
go run main.go -agent -agent-port 8081  # Машина 2  
go run main.go -agent -agent-port 8081  # Машина 3

# Сценарий: тестирование e-commerce сайта
# Агент 1 (Москва): HTTP нагрузка на каталог товаров
# Агент 2 (СПб): WebSocket подключения к чату поддержки
# Агент 3 (Екб): gRPC вызовы к сервису платежей
# Агент 4 (Казань): CPU + память для имитации фоновых процессов

# Всё управляется из одного веб-интерфейса!
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

## WebSocket и gRPC

Недавно добавил поддержку WebSocket и gRPC тестирования - оказалось очень полезно для современных приложений.

**WebSocket** крут для:
- Тестирования чат-серверов
- Симуляции множественных пользователей
- Проверки real-time приложений
- Игровых серверов

**gRPC** помогает с:
- Микросервисной архитектурой
- Health check'ами
- Тестированием streaming методов
- Проверкой производительности API

Те же паттерны нагрузки работают и тут. Подробности в [WEBSOCKET_GRPC_GUIDE.md](WEBSOCKET_GRPC_GUIDE.md).

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
- `cpu_stress_current_load` - сколько сейчас процессор нагружен
- `cpu_stress_average_load` - средняя нагрузка  
- `cpu_stress_samples_total` - количество измерений

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

### WebSocket метрики:
- `websocket_connections_total` - общее количество соединений
- `websocket_active_connections` - активные соединения сейчас
- `websocket_connections_failed_total` - неудачные подключения
- `websocket_messages_sent_total` - отправленные сообщения
- `websocket_messages_received_total` - полученные сообщения
- `websocket_connection_time_seconds` - время установки соединения
- `websocket_success_rate_percent` - процент успешных подключений

### gRPC метрики:
- `grpc_requests_total` - общее количество запросов
- `grpc_requests_success_total` - успешные запросы
- `grpc_requests_failed_total` - неудачные запросы
- `grpc_requests_per_second` - текущий RPS
- `grpc_response_time_seconds` - время ответа
- `grpc_status_codes_total` - счетчики по статус кодам
- `grpc_success_rate_percent` - процент успешных запросов

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

# WebSocket тест
docker run --rm stresspulse -websocket -websocket-url "ws://echo.websocket.org" -websocket-cps 5

# gRPC тест
docker run --rm stresspulse -grpc -grpc-addr "your-grpc-server:9000" -grpc-rps 10

# Ограничить ресурсы
docker run --cpus 0.5 --memory 512m stresspulse -cpu 50 -memory -memory-target 100

# Полное окружение с мониторингом
docker-compose up -d
```

## Kubernetes

Есть готовые конфиги в папке `helm/stresspulse`. Можно деплоить одной командой:

```bash
# Автоматический деплой
./deploy.sh

# Или руками
helm install stresspulse ./helm/stresspulse

# Поменять настройки
helm upgrade stresspulse ./helm/stresspulse --set config.cpu=80

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
- `network/` - HTTP, WebSocket и gRPC нагрузочное тестирование
- `metrics/` - интеграция с Prometheus
- `web/` - веб-интерфейс и статические файлы
- `agent/` - логика агентов и менеджер агентов
- `helm/` - конфиги для Kubernetes
- `monitoring/` - настройки Prometheus/Grafana

## Что дальше

Планирую добавить:
- Нагрузку на диск (I/O операции)
- Больше паттернов 
- ✅ ~~Web-интерфейс для управления~~ (Готово!)
- ✅ ~~Distributed режим для кластерного тестирования~~ (Готово!)
- Больше типов фейковых логов (Kubernetes, Redis, ElasticSearch)
- Автоматическое обнаружение агентов в сети
- Шаблоны нагрузочных тестов для популярных сценариев

## Известные проблемы

- На Windows иногда глючат WebSocket соединения (работаю над этим)
- gRPC с TLS может быть капризным на некоторых версиях Go
- При очень высоких RPS (>10k) может просадка производительности - см. [PERFORMANCE.md](PERFORMANCE.md)

## Оптимизация производительности

Для высоких нагрузок (>10k RPS) добавил автоматические оптимизации:

- **Автомасштабирование воркеров** - больше RPS = больше воркеров
- **Адаптивные интервалы** - при высоких RPS используются более частые тики
- **Оптимизированные connection pools** - больше соединений для HTTP/gRPC
- **Увеличенные буферы** - каналы автоматически расширяются

Подробные рекомендации по настройке системы и достижению максимальной производительности - в [PERFORMANCE.md](PERFORMANCE.md).

Теоретические максимумы на современном железе:
- HTTP: ~50k RPS 
- WebSocket: ~5k соединений/сек
- gRPC: ~30k RPS