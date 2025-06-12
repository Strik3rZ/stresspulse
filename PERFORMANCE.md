# Оптимизация производительности StressPulse

При высоких RPS (>10k) могут возникать узкие места. Вот что можно сделать для максимальной производительности.

## Автоматические оптимизации (уже в коде)

- **Автомасштабирование воркеров** - количество воркеров автоматически увеличивается с ростом целевого RPS
- **Увеличенные буферы каналов** - размер буферов теперь `targetRPS * 4`
- **Оптимизированные connection pools** - больше соединений для HTTP/gRPC
- **Адаптивные интервалы** - при RPS >5k используются более частые тики (50ms вместо 100ms)

## Настройки Go runtime

### 1. Увеличить GOMAXPROCS
```bash
export GOMAXPROCS=16  # или количество ядер + 2-4
```

### 2. Настроить garbage collector
```bash
# Менее агрессивный GC для высокой пропускной способности
export GOGC=400

# Или совсем отключить на время теста (осторожно!)
export GOGC=off
```

### 3. Запуск с оптимизациями
```bash
go build -ldflags="-s -w" .
GOMAXPROCS=16 GOGC=400 ./stresspulse -http -http-rps 15000 -http-url "http://target.com"
```

## Системные настройки

### 1. Увеличить лимиты файловых дескрипторов
```bash
# Временно для текущей сессии
ulimit -n 65536

# Постоянно в /etc/security/limits.conf
echo "* soft nofile 65536" >> /etc/security/limits.conf
echo "* hard nofile 65536" >> /etc/security/limits.conf
```

### 2. Настройки сети (Linux)
```bash
# Увеличиваем лимиты сокетов
echo 'net.core.somaxconn = 65536' >> /etc/sysctl.conf
echo 'net.ipv4.ip_local_port_range = 1024 65535' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_fin_timeout = 30' >> /etc/sysctl.conf

# Применяем
sysctl -p
```

### 3. Настройки TCP (для высоких RPS)
```bash
echo 'net.ipv4.tcp_tw_reuse = 1' >> /etc/sysctl.conf
echo 'net.ipv4.tcp_timestamps = 1' >> /etc/sysctl.conf
echo 'net.core.netdev_max_backlog = 30000' >> /etc/sysctl.conf
```

## Профилирование и отладка

### 1. Встроенное профилирование Go
```bash
# CPU профиль
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30

# Память профиль  
go tool pprof http://localhost:6060/debug/pprof/heap

# Горутины
go tool pprof http://localhost:6060/debug/pprof/goroutine
```

### 2. Включить pprof (добавить в main.go)
```go
import _ "net/http/pprof"

// В main()
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()
```

## Оптимизация по протоколам

### HTTP тестирование
```bash
# Для максимальной производительности
./stresspulse -http -http-rps 20000 \
  -http-url "http://target.com/fast-endpoint" \
  -http-method GET \
  -workers 0  # авто-определение
```

**Советы:**
- GET запросы быстрее POST
- Маленькие URL работают быстрее
- Избегайте сложных заголовков при высоких RPS

### WebSocket тестирование
```bash
# Высокие CPS
./stresspulse -websocket -websocket-cps 1000 \
  -websocket-url "ws://target.com/ws" \
  -websocket-message-interval 5s  # не слишком часто
```

**Ограничения:**
- WebSocket соединения потребляют много памяти
- При CPS >500 следите за лимитами дескрипторов
- Большие интервалы сообщений = выше CPS

### gRPC тестирование  
```bash
# Оптимизированный gRPC
./stresspulse -grpc -grpc-rps 10000 \
  -grpc-addr "target.com:443" \
  -grpc-method health_check  # самый быстрый
```

**Оптимизации:**
- health_check методы самые легковесные
- Избегайте TLS если не нужен
- Увеличивайте connection pool автоматически

## Мониторинг производительности

### 1. Системные ресурсы
```bash
# CPU и память
htop

# Сетевая активность
iftop

# Дескрипторы файлов
lsof -p $(pgrep stresspulse) | wc -l
```

### 2. Метрики приложения
```bash
# Текущая статистика
curl http://localhost:9090/metrics | grep -E "(rps|connections|response_time)"
```

### 3. Признаки проблем производительности
- **Растущее время ответа** - перегружен целевой сервер или сеть
- **Много failed requests** - достигли лимитов
- **Низкий actual RPS при высоком target** - узкое место в StressPulse
- **Высокое потребление памяти** - проблемы с GC или утечки

## Распределенное тестирование

Для RPS >50k рекомендуется запускать несколько экземпляров:

```bash
# Машина 1
./stresspulse -http -http-rps 15000 -http-url "http://target.com" -metrics-port 9091

# Машина 2  
./stresspulse -http -http-rps 15000 -http-url "http://target.com" -metrics-port 9092

# Машина 3
./stresspulse -http -http-rps 15000 -http-url "http://target.com" -metrics-port 9093
```

Общий RPS = 45k, но нагрузка распределена.

## Известные лимиты

### Теоретические максимумы (на современном железе):
- **HTTP**: ~50k RPS с одного экземпляра
- **WebSocket**: ~5k CPS (connections per second)  
- **gRPC**: ~30k RPS

### Практические рекомендации:
- **Начинайте с 1k RPS** и увеличивайте постепенно
- **Мониторьте целевой сервер** - он может быть узким местом
- **Используйте несколько машин** для экстремальных нагрузок

## Troubleshooting

### "Too many open files"
```bash
ulimit -n 65536
```

### "Connection refused"
```bash
# Увеличиваем лимиты системы
echo 'net.core.somaxconn = 65536' >> /etc/sysctl.conf
sysctl -p
```

### Высокое потребление памяти
```bash
# Запуск с ограничением памяти
GOGC=100 ./stresspulse -http -http-rps 10000
```

### Низкая производительность на Windows
```bash
# Windows плохо работает с высокими RPS
# Рекомендуется Linux или WSL2
```

---

При правильной настройке StressPulse может генерировать нагрузку в десятки тысяч RPS. Главное - следить за ресурсами системы и постепенно увеличивать нагрузку. 