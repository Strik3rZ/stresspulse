package config

import (
	"flag"
	"time"
)

type Config struct {
	TargetCPUPercent  float64
	Duration          time.Duration
	DriftAmplitude    float64
	DriftPeriod       time.Duration
	NumWorkers        int
	LogLevel          string
	MetricsEnabled    bool
	MetricsPort       int
	PatternType       string
	ProfilePath       string
	SaveProfile       bool
	FakeLogsEnabled   bool
	FakeLogsType      string
	FakeLogsInterval  time.Duration

	MemoryEnabled     bool
	MemoryTargetMB    int
	MemoryPattern     string
	MemoryInterval    time.Duration

	HTTPEnabled       bool
	HTTPTargetURL     string
	HTTPTargetRPS     int
	HTTPPattern       string
	HTTPMethod        string
	HTTPTimeout       time.Duration
	HTTPHeaders       string
	HTTPBody          string

	WebSocketEnabled         bool
	WebSocketTargetURL       string
	WebSocketTargetCPS       int
	WebSocketPattern         string
	WebSocketMessageInterval time.Duration
	WebSocketMessageSize     int
	WebSocketHeaders         string

	GRPCEnabled      bool
	GRPCTargetAddr   string
	GRPCTargetRPS    int
	GRPCPattern      string
	GRPCServiceName  string
	GRPCMethodType   string
	GRPCUseSecure    bool
	GRPCMetadata     string

	WebEnabled       bool
	WebPort          int

	AgentMode bool `json:"agent_mode"`
	AgentPort int  `json:"agent_port"`
}

func NewConfig() *Config {
	return &Config{
		TargetCPUPercent: 50,
		Duration:         0,
		DriftAmplitude:   20,
		DriftPeriod:      30 * time.Second,
		NumWorkers:       0,
		LogLevel:         "info",
		MetricsEnabled:   false,
		MetricsPort:      9090,
		PatternType:      "sine",
		ProfilePath:      "profile.json",
		SaveProfile:      false,
		FakeLogsEnabled:  false,
		FakeLogsType:     "java",
		FakeLogsInterval: 1 * time.Second,
		MemoryEnabled:    false,
		MemoryTargetMB:   100,
		MemoryPattern:    "constant",
		MemoryInterval:   2 * time.Second,
		HTTPEnabled:      false,
		HTTPTargetURL:    "http://localhost:8080/health",
		HTTPTargetRPS:    10,
		HTTPPattern:      "constant",
		HTTPMethod:       "GET",
		HTTPTimeout:      5 * time.Second,
		HTTPHeaders:      "",
		HTTPBody:         "",

		WebSocketEnabled:         false,
		WebSocketTargetURL:       "ws://localhost:8080/ws",
		WebSocketTargetCPS:       5,
		WebSocketPattern:         "constant",
		WebSocketMessageInterval: 1 * time.Second,
		WebSocketMessageSize:     256,
		WebSocketHeaders:         "",

		GRPCEnabled:     false,
		GRPCTargetAddr:  "localhost:9000",
		GRPCTargetRPS:   10,
		GRPCPattern:     "constant",
		GRPCServiceName: "",
		GRPCMethodType:  "health_check",
		GRPCUseSecure:   false,
		GRPCMetadata:    "",

		WebEnabled:      false,
		WebPort:         8080,

		AgentMode: false,
		AgentPort: 8081,
	}
}

func (c *Config) ParseFlags() {
	flag.Float64Var(&c.TargetCPUPercent, "cpu", c.TargetCPUPercent, "Целевой процент нагрузки CPU (0-100)")
	flag.DurationVar(&c.Duration, "duration", c.Duration, "Длительность теста (0 для бесконечного выполнения)")
	flag.Float64Var(&c.DriftAmplitude, "drift", c.DriftAmplitude, "Амплитуда дрейфа нагрузки в процентах")
	flag.DurationVar(&c.DriftPeriod, "period", c.DriftPeriod, "Период дрейфа нагрузки")
	flag.IntVar(&c.NumWorkers, "workers", c.NumWorkers, "Количество горутин-воркеров (0 для автоматического)")
	flag.StringVar(&c.LogLevel, "log-level", c.LogLevel, "Уровень логирования (debug, info, warn, error)")
	flag.BoolVar(&c.MetricsEnabled, "metrics", c.MetricsEnabled, "Включение сбора метрик")
	flag.IntVar(&c.MetricsPort, "metrics-port", c.MetricsPort, "Порт для сервера метрик (1024-65535)")
	flag.StringVar(&c.PatternType, "pattern", c.PatternType, "Тип паттерна нагрузки (sine, square, sawtooth, random)")
	flag.StringVar(&c.ProfilePath, "profile", c.ProfilePath, "Путь для сохранения/загрузки профиля CPU")
	flag.BoolVar(&c.SaveProfile, "save-profile", c.SaveProfile, "Сохранение профиля CPU после теста")
	flag.BoolVar(&c.FakeLogsEnabled, "fake-logs", c.FakeLogsEnabled, "Включение генерации фейковых логов")
	flag.StringVar(&c.FakeLogsType, "fake-logs-type", c.FakeLogsType, "Тип фейковых логов (java, web, microservice, database, ecommerce)")
	flag.DurationVar(&c.FakeLogsInterval, "fake-logs-interval", c.FakeLogsInterval, "Интервал генерации фейковых логов")
	flag.BoolVar(&c.MemoryEnabled, "memory", c.MemoryEnabled, "Включение нагрузки на память")
	flag.IntVar(&c.MemoryTargetMB, "memory-target", c.MemoryTargetMB, "Целевое количество памяти в MB")
	flag.StringVar(&c.MemoryPattern, "memory-pattern", c.MemoryPattern, "Паттерн использования памяти (constant, leak, spike, cycle, random)")
	flag.DurationVar(&c.MemoryInterval, "memory-interval", c.MemoryInterval, "Интервал операций с памятью")
	flag.BoolVar(&c.HTTPEnabled, "http", c.HTTPEnabled, "Включение HTTP нагрузочного тестирования")
	flag.StringVar(&c.HTTPTargetURL, "http-url", c.HTTPTargetURL, "URL для HTTP нагрузочного тестирования")
	flag.IntVar(&c.HTTPTargetRPS, "http-rps", c.HTTPTargetRPS, "Целевое количество запросов в секунду")
	flag.StringVar(&c.HTTPPattern, "http-pattern", c.HTTPPattern, "Паттерн HTTP нагрузки (constant, spike, cycle, ramp, random)")
	flag.StringVar(&c.HTTPMethod, "http-method", c.HTTPMethod, "HTTP метод (GET, POST, PUT, DELETE)")
	flag.DurationVar(&c.HTTPTimeout, "http-timeout", c.HTTPTimeout, "Таймаут HTTP запросов")
	flag.StringVar(&c.HTTPHeaders, "http-headers", c.HTTPHeaders, "HTTP заголовки в формате 'Key1:Value1,Key2:Value2'")
	flag.StringVar(&c.HTTPBody, "http-body", c.HTTPBody, "Тело HTTP запроса")
	
	flag.BoolVar(&c.WebSocketEnabled, "websocket", c.WebSocketEnabled, "Включение WebSocket нагрузочного тестирования")
	flag.StringVar(&c.WebSocketTargetURL, "websocket-url", c.WebSocketTargetURL, "URL для WebSocket соединений")
	flag.IntVar(&c.WebSocketTargetCPS, "websocket-cps", c.WebSocketTargetCPS, "Целевое количество соединений в секунду")
	flag.StringVar(&c.WebSocketPattern, "websocket-pattern", c.WebSocketPattern, "Паттерн WebSocket нагрузки (constant, spike, cycle, ramp, random)")
	flag.DurationVar(&c.WebSocketMessageInterval, "websocket-message-interval", c.WebSocketMessageInterval, "Интервал отправки сообщений")
	flag.IntVar(&c.WebSocketMessageSize, "websocket-message-size", c.WebSocketMessageSize, "Размер сообщений в байтах")
	flag.StringVar(&c.WebSocketHeaders, "websocket-headers", c.WebSocketHeaders, "WebSocket заголовки в формате 'Key1:Value1,Key2:Value2'")
	
	flag.BoolVar(&c.GRPCEnabled, "grpc", c.GRPCEnabled, "Включение gRPC нагрузочного тестирования")
	flag.StringVar(&c.GRPCTargetAddr, "grpc-addr", c.GRPCTargetAddr, "Адрес gRPC сервера")
	flag.IntVar(&c.GRPCTargetRPS, "grpc-rps", c.GRPCTargetRPS, "Целевое количество запросов в секунду")
	flag.StringVar(&c.GRPCPattern, "grpc-pattern", c.GRPCPattern, "Паттерн gRPC нагрузки (constant, spike, cycle, ramp, random)")
	flag.StringVar(&c.GRPCServiceName, "grpc-service", c.GRPCServiceName, "Имя gRPC сервиса для health check")
	flag.StringVar(&c.GRPCMethodType, "grpc-method", c.GRPCMethodType, "Тип gRPC метода (health_check, unary, server_stream, client_stream, bidi_stream)")
	flag.BoolVar(&c.GRPCUseSecure, "grpc-secure", c.GRPCUseSecure, "Использовать TLS для gRPC соединений")
	flag.StringVar(&c.GRPCMetadata, "grpc-metadata", c.GRPCMetadata, "gRPC метаданные в формате 'Key1:Value1,Key2:Value2'")
	
	flag.BoolVar(&c.WebEnabled, "web", c.WebEnabled, "Включить веб-интерфейс управления")
	flag.IntVar(&c.WebPort, "web-port", c.WebPort, "Порт веб-интерфейса (1024-65535)")
	
	flag.BoolVar(&c.AgentMode, "agent", c.AgentMode, "Run in agent mode")
	flag.IntVar(&c.AgentPort, "agent-port", c.AgentPort, "Port for agent mode")
	
	flag.Parse()
}

func (c *Config) Validate() error {
	if c.TargetCPUPercent < 0 || c.TargetCPUPercent > 100 {
		return ErrInvalidCPUPercentage
	}
	if c.DriftAmplitude < 0 {
		return ErrInvalidDriftAmplitude
	}
	if c.DriftPeriod <= 0 {
		return ErrInvalidDriftPeriod
	}
	if c.NumWorkers < 0 {
		return ErrInvalidWorkerCount
	}
	if c.PatternType != "sine" && c.PatternType != "square" && c.PatternType != "sawtooth" && c.PatternType != "random" {
		return ErrInvalidPatternType
	}
	if c.MetricsEnabled && (c.MetricsPort < 1024 || c.MetricsPort > 65535) {
		return ErrInvalidMetricsPort
	}
	if c.LogLevel != "debug" && c.LogLevel != "info" && c.LogLevel != "warn" && c.LogLevel != "error" {
		return ErrInvalidLogLevel
	}
	if c.FakeLogsEnabled {
		validTypes := []string{"java", "web", "microservice", "database", "ecommerce", "generic"}
		valid := false
		for _, validType := range validTypes {
			if c.FakeLogsType == validType {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidFakeLogsType
		}
	}
	if c.MemoryEnabled {
		if c.MemoryTargetMB <= 0 {
			return ErrInvalidMemoryTarget
		}
		validPatterns := []string{"constant", "leak", "spike", "cycle", "random"}
		valid := false
		for _, pattern := range validPatterns {
			if c.MemoryPattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidMemoryPattern
		}
	}
	if c.HTTPEnabled {
		if c.HTTPTargetURL == "" {
			return ErrInvalidHTTPURL
		}
		if c.HTTPTargetRPS <= 0 {
			return ErrInvalidHTTPRPS
		}
		validMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
		valid := false
		for _, method := range validMethods {
			if c.HTTPMethod == method {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidHTTPMethod
		}
		validPatterns := []string{"constant", "spike", "cycle", "ramp", "random"}
		valid = false
		for _, pattern := range validPatterns {
			if c.HTTPPattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidHTTPPattern
		}
	}
	if c.WebSocketEnabled {
		if c.WebSocketTargetURL == "" {
			return ErrInvalidWebSocketURL
		}
		if c.WebSocketTargetCPS <= 0 {
			return ErrInvalidWebSocketCPS
		}
		validPatterns := []string{"constant", "spike", "cycle", "ramp", "random"}
		valid := false
		for _, pattern := range validPatterns {
			if c.WebSocketPattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidWebSocketPattern
		}
	}
	if c.GRPCEnabled {
		if c.GRPCTargetAddr == "" {
			return ErrInvalidGRPCAddress
		}
		if c.GRPCTargetRPS <= 0 {
			return ErrInvalidGRPCRPS
		}
		validPatterns := []string{"constant", "spike", "cycle", "ramp", "random"}
		valid := false
		for _, pattern := range validPatterns {
			if c.GRPCPattern == pattern {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidGRPCPattern
		}
		validMethods := []string{"health_check", "unary", "server_stream", "client_stream", "bidi_stream"}
		valid = false
		for _, method := range validMethods {
			if c.GRPCMethodType == method {
				valid = true
				break
			}
		}
		if !valid {
			return ErrInvalidGRPCMethodType
		}
	}
	
	// Web Interface validation
	if c.WebEnabled {
		if c.WebPort < 1024 || c.WebPort > 65535 {
			return ErrInvalidWebPort
		}
	}

	if c.AgentMode {
		if c.AgentPort < 1024 || c.AgentPort > 65535 {
			return ErrInvalidAgentPort
		}
	}
	
	return nil
} 