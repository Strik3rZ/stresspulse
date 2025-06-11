package config

import "errors"

var (
	ErrInvalidCPUPercentage = errors.New("CPU percentage must be between 0 and 100")
	ErrInvalidDriftAmplitude = errors.New("drift amplitude must be non-negative")
	ErrInvalidDriftPeriod = errors.New("drift period must be positive")
	ErrInvalidWorkerCount = errors.New("worker count must be non-negative")
	ErrInvalidPatternType = errors.New("invalid pattern type")
	ErrInvalidMetricsPort = errors.New("metrics port must be between 1024 and 65535")
	ErrInvalidLogLevel = errors.New("invalid log level")
	ErrInvalidFakeLogsType = errors.New("invalid fake logs type")
	ErrInvalidFakeLogsInterval = errors.New("fake logs interval must be positive")
	ErrInvalidMemoryTarget = errors.New("memory target must be positive")
	ErrInvalidMemoryPattern = errors.New("invalid memory pattern")
	ErrInvalidMemoryInterval = errors.New("memory interval must be positive")
	ErrInvalidHTTPURL = errors.New("HTTP URL cannot be empty")
	ErrInvalidHTTPRPS = errors.New("HTTP RPS must be positive")
	ErrInvalidHTTPPattern = errors.New("invalid HTTP pattern")
	ErrInvalidHTTPMethod = errors.New("invalid HTTP method")
	ErrInvalidHTTPTimeout = errors.New("HTTP timeout must be positive")
) 