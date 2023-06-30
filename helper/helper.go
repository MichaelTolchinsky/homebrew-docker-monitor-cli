package helper

import (
	"fmt"

	"github.com/docker/docker/api/types"
)

// Helper functions to calculate different stats

func CalculateCPUPercentage(stats types.Stats) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	cpuPercent := (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	return cpuPercent
}

func CalculateMemUsage(stats types.Stats) string {
	memUsage := stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"]
	return formatBytes(memUsage)
}

func CalculateMemLimit(stats types.Stats) string {
	return formatBytes(stats.MemoryStats.Limit)
}

func CalculateMemPercentage(stats types.Stats) float64 {
	memPercent := float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	return memPercent
}

func CalculateBlockInput(stats types.Stats) string {
	blockInput := calculateBlkioValue(stats.BlkioStats.IoServiceBytesRecursive, "Read")
	return formatBytes(blockInput)
}

func CalculateBlockOutput(stats types.Stats) string {
	blockOutput := calculateBlkioValue(stats.BlkioStats.IoServiceBytesRecursive, "Write")
	return formatBytes(blockOutput)
}

func calculateBlkioValue(blkioStats []types.BlkioStatEntry, opType string) uint64 {
	for _, blkioStat := range blkioStats {
		if blkioStat.Op == opType {
			return blkioStat.Value
		}
	}
	return 0
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
