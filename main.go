package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/olekukonko/tablewriter"
)

func main() {
	// Create a new Docker client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	// Get a list of running containers
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	// Prepare the table writer
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"CONTAINER ID", "NAME", "CPU %", "MEM USAGE / LIMIT", "MEM %", "BLOCK I/O", "PIDS"})

	// Create a wait group to wait for goroutines to complete
	var wg sync.WaitGroup

	// Iterate over the containers and read their stats
	for _, container := range containers {
		wg.Add(1) // Increment the wait group counter

		go func(container types.Container) {
			defer wg.Done() // Signal the wait group when the goroutine is done

			stats, err := cli.ContainerStats(context.Background(), container.ID, false)
			if err != nil {
				fmt.Printf("Error retrieving stats for container %s: %s\n", container.ID, err.Error())
				return
			}
			defer stats.Body.Close()

			var result types.Stats
			decoder := json.NewDecoder(stats.Body)
			for {
				if err := decoder.Decode(&result); err != nil {
					break
				}

				cpuUsage := fmt.Sprintf("%.2f%%", calculateCPUPercentage(result))
				memUsage := fmt.Sprintf("%s / %s", calculateMemUsage(result), calculateMemLimit(result))
				memPercent := fmt.Sprintf("%.2f%%", calculateMemPercentage(result))
				blockIO := fmt.Sprintf("%s / %s", calculateBlockInput(result), calculateBlockOutput(result))
				pids := fmt.Sprintf("%d", result.PidsStats.Current)

				// Add the stats to the table
				table.Append([]string{container.ID[:12], container.Names[0], cpuUsage, memUsage, memPercent, blockIO, pids})
			}
		}(container)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Render the table
	table.Render()

	// wait indefinitly
	select {}
}

// Helper functions to calculate different stats

func calculateCPUPercentage(stats types.Stats) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage) - float64(stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage) - float64(stats.PreCPUStats.SystemUsage)
	cpuPercent := (cpuDelta / systemDelta) * float64(len(stats.CPUStats.CPUUsage.PercpuUsage)) * 100.0
	return cpuPercent
}

func calculateMemUsage(stats types.Stats) string {
	memUsage := stats.MemoryStats.Usage - stats.MemoryStats.Stats["cache"]
	return formatBytes(memUsage)
}

func calculateMemLimit(stats types.Stats) string {
	return formatBytes(stats.MemoryStats.Limit)
}

func calculateMemPercentage(stats types.Stats) float64 {
	memPercent := float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	return memPercent
}

func calculateBlockInput(stats types.Stats) string {
	blockInput := calculateBlkioValue(stats.BlkioStats.IoServiceBytesRecursive, "Read")
	return formatBytes(blockInput)
}

func calculateBlockOutput(stats types.Stats) string {
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
