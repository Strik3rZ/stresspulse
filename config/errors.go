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
) 