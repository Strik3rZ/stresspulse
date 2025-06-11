# StressPulse

Простой генератор нагрузки на CPU. Написал потому что надоело возиться с примитивными утилитами типа `stress` - хотелось что-то поумнее и с нормальным мониторингом.

Главная фишка - это "живая" нагрузка. Программа не просто жрёт процессор на одном уровне, а создаёт реалистичные паттерны с колебаниями, как в настоящих приложениях.

## Зачем это нужно

Иногда хочется протестировать как система ведёт себя под нагрузкой. Обычные стресс-тесты создают постоянную нагрузку, а в реальности она всегда скачет. Поэтому сделал инструмент, который умеет:

- Плавно менять нагрузку по разным паттернам
- Собирать метрики для Prometheus/Grafana  
- Работать в Docker и Kubernetes
- Не сожрать весь сервер случайно
- **Генерировать реалистичные логи как у настоящих приложений**

## Как попробовать

Самый быстрый способ:

```bash
# Скачай зависимости
go mod tidy

# Запусти на 50% CPU
go run main.go

# С фейковыми логами Java приложения
go run main.go -cpu 70 -fake-logs -fake-logs-type java

# Веб-сервер с активными логами
go run main.go -cpu 50 -fake-logs -fake-logs-type web -fake-logs-interval 500ms
```

Если есть Docker:

```bash
# Собери и запусти
docker build -t stresspulse .
docker run --rm stresspulse -cpu 70 -metrics -fake-logs
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

# Имитация Java приложения под нагрузкой
go run main.go -cpu 60 -fake-logs -fake-logs-type java -fake-logs-interval 800ms

# Имитация микросервисной архитектуры
go run main.go -cpu 45 -fake-logs -fake-logs-type microservice -fake-logs-interval 300ms

# Стресс-тест базы данных с логами
go run main.go -cpu 70 -fake-logs -fake-logs-type database -fake-logs-interval 2s -duration 1h

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

- `cpu_usage_percent` - сколько сейчас процессор нагружен
- `cpu_target_percent` - сколько хотели нагрузить  
- `pattern_value` - текущее значение паттерна
- `workers_count` - сколько потоков работает

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

# Ограничить ресурсы (чтобы не убить хост)
docker run --cpus 0.5 --memory 128m stresspulse -cpu 50

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
- `metrics/` - интеграция с Prometheus
- `helm/` - конфиги для Kubernetes
- `monitoring/` - настройки Prometheus/Grafana

## Что дальше

Планирую добавить:
- Нагрузку на память и диск
- Больше паттернов 
- Web-интерфейс для управления
- Distributed режим для кластерного тестирования
- Больше типов фейковых логов (Kubernetes, Redis, ElasticSearch)
