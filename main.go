package main

import (
	"context"
	"docker-monitor-cli/helper"
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

				cpuUsage := fmt.Sprintf("%.2f%%", helper.CalculateCPUPercentage(result))
				memUsage := fmt.Sprintf("%s / %s", helper.CalculateMemUsage(result), helper.CalculateMemLimit(result))
				memPercent := fmt.Sprintf("%.2f%%", helper.CalculateMemPercentage(result))
				blockIO := fmt.Sprintf("%s / %s", helper.CalculateBlockInput(result), helper.CalculateBlockOutput(result))
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
