package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func (m *model) getStats(id string) (string, string) {
	resp, err := m.docker.ContainerStatsOneShot(context.Background(), id)
	if err != nil {
		return "0.00%", "0.0MB"
	}
	defer resp.Body.Close()

	var v container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return "N/A", "N/A"
	}

	cpuUsage := v.CPUStats.CPUUsage.TotalUsage
	systemUsage := v.CPUStats.SystemUsage

	cpuPct := 0.0
	cpuDelta := float64(cpuUsage) - float64(m.prevCPU[id])
	systemDelta := float64(systemUsage) - float64(m.prevSystem[id])

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpus := float64(v.CPUStats.OnlineCPUs)
		if cpus == 0 {
			cpus = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
		}
		cpuPct = (cpuDelta / systemDelta) * cpus * 100.0
	}

	m.prevCPU[id] = cpuUsage
	m.prevSystem[id] = systemUsage

	memUsage := float64(v.MemoryStats.Usage) / 1024 / 1024
	return fmt.Sprintf("%.2f%%", cpuPct), fmt.Sprintf("%.1fMB", memUsage)
}

func (m model) controlContainer(id string, action string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		var err error
		switch action {
		case "start":
			err = m.docker.ContainerStart(ctx, id, container.StartOptions{})
		case "stop":
			err = m.docker.ContainerStop(ctx, id, container.StopOptions{})
		case "restart":
			err = m.docker.ContainerRestart(ctx, id, container.StopOptions{}) // Default timeout
		case "pause":
			err = m.docker.ContainerPause(ctx, id)
		case "unpause":
			err = m.docker.ContainerUnpause(ctx, id)
		case "remove":
			err = m.docker.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
		case "prune":
			_, err = m.docker.ContainersPrune(ctx, filters.Args{})
		}
		if err != nil {
			return logLineMsg(fmt.Sprintf("\nError performing %s: %v", action, err))
		}
		return fetchContainers(m.docker)()
	}
}

func (m model) waitForLogs() tea.Cmd {
	return func() tea.Msg { return logLineMsg(<-m.logChan) }
}

func doTick() tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func startLogging(ctx context.Context, cli *client.Client, id string, logChan chan string) {
	out, err := cli.ContainerLogs(ctx, id, container.LogsOptions{ShowStdout: true, ShowStderr: true, Follow: true, Tail: "20"})
	if err != nil {
		return
	}
	defer out.Close()
	cw := channelWriter{logChan: logChan}
	stdcopy.StdCopy(cw, cw, out)
}
