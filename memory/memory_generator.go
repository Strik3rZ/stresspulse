package memory

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"stresspulse/logger"
)

type MemoryGenerator struct {
	targetMemoryMB   int
	pattern          string
	interval         time.Duration
	enabled          bool
	allocatedBlocks  [][]byte
	mutex            sync.Mutex
	ctx              context.Context
	cancel           context.CancelFunc
	stats            *MemoryStats
}

type MemoryStats struct {
	AllocatedMB     int
	TotalAllocated  int64
	TotalReleased   int64
	AllocationCount int64
	StartTime       time.Time
}

func NewMemoryGenerator(targetMemoryMB int, pattern string, interval time.Duration) *MemoryGenerator {
	return &MemoryGenerator{
		targetMemoryMB:  targetMemoryMB,
		pattern:         pattern,
		interval:        interval,
		enabled:         false,
		allocatedBlocks: make([][]byte, 0),
		stats: &MemoryStats{
			StartTime: time.Now(),
		},
	}
}

func (mg *MemoryGenerator) Start(ctx context.Context) {
	if mg.enabled {
		return
	}

	mg.enabled = true
	mg.ctx = ctx
	
	logger.Info("Starting memory stress generator: %dMB target, pattern: %s", mg.targetMemoryMB, mg.pattern)
	
	go mg.generateMemoryLoad()
}

func (mg *MemoryGenerator) Stop() {
	if !mg.enabled {
		return
	}

	mg.enabled = false
	
	mg.mutex.Lock()
	mg.allocatedBlocks = nil
	runtime.GC()
	mg.mutex.Unlock()
	
	logger.Info("Memory stress generator stopped")
}

func (mg *MemoryGenerator) generateMemoryLoad() {
	ticker := time.NewTicker(mg.interval)
	defer ticker.Stop()

	for {
		select {
		case <-mg.ctx.Done():
			return
		case <-ticker.C:
			mg.executePattern()
		}
	}
}

func (mg *MemoryGenerator) executePattern() {
	switch mg.pattern {
	case "constant":
		mg.constantAllocation()
	case "leak":
		mg.memoryLeak()
	case "spike":
		mg.memorySpike()
	case "cycle":
		mg.memoryCycle()
	case "random":
		mg.randomAllocation()
	default:
		mg.constantAllocation()
	}
}

func (mg *MemoryGenerator) constantAllocation() {
	mg.mutex.Lock()
	defer mg.mutex.Unlock()

	currentMB := mg.getCurrentAllocatedMB()
	
	if currentMB < mg.targetMemoryMB {
		toAllocate := mg.targetMemoryMB - currentMB
		mg.allocateMemory(toAllocate)
	} else if currentMB > mg.targetMemoryMB {
		mg.releaseExcessMemory(currentMB - mg.targetMemoryMB)
	}
}

func (mg *MemoryGenerator) memoryLeak() {
	mg.mutex.Lock()
	defer mg.mutex.Unlock()

	leakSize := rand.Intn(5) + 1
	mg.allocateMemory(leakSize)
	
	if rand.Intn(10) == 0 && len(mg.allocatedBlocks) > 10 {
		toRelease := len(mg.allocatedBlocks) / 4
		mg.releaseBlocks(toRelease)
	}
}

func (mg *MemoryGenerator) memorySpike() {
	mg.mutex.Lock()
	defer mg.mutex.Unlock()

	currentMB := mg.getCurrentAllocatedMB()
	
	if rand.Intn(5) == 0 {
		spikeMultiplier := rand.Intn(2) + 2
		targetSpike := mg.targetMemoryMB * spikeMultiplier
		if currentMB < targetSpike {
			mg.allocateMemory(targetSpike - currentMB)
		}
	} else {
		if currentMB > mg.targetMemoryMB {
			mg.releaseExcessMemory(currentMB - mg.targetMemoryMB)
		}
	}
}

func (mg *MemoryGenerator) memoryCycle() {
	mg.mutex.Lock()
	defer mg.mutex.Unlock()

	currentMB := mg.getCurrentAllocatedMB()
	
	elapsedSeconds := int(time.Since(mg.stats.StartTime).Seconds())
	cyclePosition := (elapsedSeconds / 30) % 4 // 30-секундные фазы
	
	var targetForPhase int
	switch cyclePosition {
	case 0:
		targetForPhase = mg.targetMemoryMB / 4
	case 1:
		targetForPhase = mg.targetMemoryMB
	case 2:
		targetForPhase = mg.targetMemoryMB / 2
	case 3:
		targetForPhase = mg.targetMemoryMB / 8
	}
	
	if currentMB < targetForPhase {
		mg.allocateMemory(targetForPhase - currentMB)
	} else if currentMB > targetForPhase {
		mg.releaseExcessMemory(currentMB - targetForPhase)
	}
}

func (mg *MemoryGenerator) randomAllocation() {
	mg.mutex.Lock()
	defer mg.mutex.Unlock()

	action := rand.Intn(3)
	
	switch action {
	case 0:
		size := rand.Intn(20) + 1
		mg.allocateMemory(size)
	case 1:
		if len(mg.allocatedBlocks) > 0 {
			toRelease := rand.Intn(len(mg.allocatedBlocks)/2 + 1)
			mg.releaseBlocks(toRelease)
		}
	case 2:
	}
}

func (mg *MemoryGenerator) allocateMemory(sizeMB int) {
	if sizeMB <= 0 {
		return
	}

	for i := 0; i < sizeMB; i++ {
		block := make([]byte, 1024*1024) // 1MB
		
		for j := 0; j < len(block); j += 1024 {
			end := j + 1024
			if end > len(block) {
				end = len(block)
			}
			for k := j; k < end; k++ {
				block[k] = byte(rand.Intn(256))
			}
		}
		
		mg.allocatedBlocks = append(mg.allocatedBlocks, block)
		mg.stats.TotalAllocated += int64(len(block))
		mg.stats.AllocationCount++
	}
	
	mg.stats.AllocatedMB = mg.getCurrentAllocatedMB()
	logger.Debug("Allocated %dMB, total: %dMB", sizeMB, mg.stats.AllocatedMB)
}

func (mg *MemoryGenerator) releaseBlocks(count int) {
	if count <= 0 || len(mg.allocatedBlocks) == 0 {
		return
	}

	if count > len(mg.allocatedBlocks) {
		count = len(mg.allocatedBlocks)
	}

	releasedSize := int64(0)
	for i := 0; i < count; i++ {
		if len(mg.allocatedBlocks) > 0 {
			lastIndex := len(mg.allocatedBlocks) - 1
			releasedSize += int64(len(mg.allocatedBlocks[lastIndex]))
			mg.allocatedBlocks[lastIndex] = nil // Помогаем GC
			mg.allocatedBlocks = mg.allocatedBlocks[:lastIndex]
		}
	}

	mg.stats.TotalReleased += releasedSize
	mg.stats.AllocatedMB = mg.getCurrentAllocatedMB()
	
	runtime.GC()
	
	logger.Debug("Released %d blocks (%.1fMB), total: %dMB", count, float64(releasedSize)/(1024*1024), mg.stats.AllocatedMB)
}

func (mg *MemoryGenerator) releaseExcessMemory(excessMB int) {
	blocksToRelease := excessMB
	if blocksToRelease > len(mg.allocatedBlocks) {
		blocksToRelease = len(mg.allocatedBlocks)
	}
	
	mg.releaseBlocks(blocksToRelease)
}

func (mg *MemoryGenerator) getCurrentAllocatedMB() int {
	totalBytes := 0
	for _, block := range mg.allocatedBlocks {
		if block != nil {
			totalBytes += len(block)
		}
	}
	return totalBytes / (1024 * 1024)
}

func (mg *MemoryGenerator) GetStats() *MemoryStats {
	mg.mutex.Lock()
	defer mg.mutex.Unlock()
	
	mg.stats.AllocatedMB = mg.getCurrentAllocatedMB()
	
	return &MemoryStats{
		AllocatedMB:     mg.stats.AllocatedMB,
		TotalAllocated:  mg.stats.TotalAllocated,
		TotalReleased:   mg.stats.TotalReleased,
		AllocationCount: mg.stats.AllocationCount,
		StartTime:       mg.stats.StartTime,
	}
}

func GetSystemMemoryStats() (allocated, totalAlloc, sys uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc, m.TotalAlloc, m.Sys
} 