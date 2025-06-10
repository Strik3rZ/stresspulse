package patterns

import (
	"math"
	"math/rand"
	"time"
)

type Config struct {
	BaseLoad       float64
	DriftAmplitude float64
	DriftPeriod    int
	NoiseAmplitude float64
	Steps          int
}

type Pattern interface {
	GetLoad(elapsed time.Duration, baseLoad, amplitude float64) float64
}

type SinePattern struct{}

func (p *SinePattern) GetLoad(elapsed time.Duration, baseLoad, amplitude float64) float64 {
	drift := math.Sin(2*math.Pi*float64(elapsed)/float64(30*time.Second)) * amplitude
	return baseLoad + drift
}

type SquarePattern struct{}

func (p *SquarePattern) GetLoad(elapsed time.Duration, baseLoad, amplitude float64) float64 {
	period := 30 * time.Second
	halfPeriod := period / 2
	if elapsed%(period) < halfPeriod {
		return baseLoad + amplitude
	}
	return baseLoad - amplitude
}

type SawtoothPattern struct{}

func (p *SawtoothPattern) GetLoad(elapsed time.Duration, baseLoad, amplitude float64) float64 {
	period := 30 * time.Second
	position := float64(elapsed%(period)) / float64(period)
	return baseLoad + (position*2-1)*amplitude
}

type RandomPattern struct {
	lastValue float64
	lastTime  time.Time
	rnd       *rand.Rand
}

func NewRandomPattern() *RandomPattern {
	return &RandomPattern{
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (p *RandomPattern) GetLoad(elapsed time.Duration, baseLoad, amplitude float64) float64 {
	if p.lastTime.IsZero() {
		p.lastTime = time.Now()
		p.lastValue = baseLoad
		return baseLoad
	}

	if elapsed%(5*time.Second) < time.Second {
		target := baseLoad + (p.rnd.Float64()*2-1)*amplitude
		p.lastValue = p.lastValue + (target-p.lastValue)*0.1
	}
	return p.lastValue
}

func NewPattern(patternType string) Pattern {
	switch patternType {
	case "sine":
		return &SinePattern{}
	case "square":
		return &SquarePattern{}
	case "sawtooth":
		return &SawtoothPattern{}
	case "random":
		return NewRandomPattern()
	default:
		return &SinePattern{}
	}
}

func generateDriftPattern(config *Config) []float64 {
	rand.Seed(time.Now().UnixNano())
	pattern := make([]float64, config.Steps)
	
	for i := 0; i < config.Steps; i++ {
		drift := config.DriftAmplitude * math.Sin(2*math.Pi*float64(i)/float64(config.DriftPeriod))
		
		noise := rand.Float64() * config.NoiseAmplitude
		
		value := config.BaseLoad + drift + noise
		pattern[i] = math.Max(0, math.Min(100, value))
	}
	
	return pattern
} 