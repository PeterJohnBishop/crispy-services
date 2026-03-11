package manager

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func fetchContainers(cli *client.Client) tea.Cmd {
	return func() tea.Msg {
		containers, _ := cli.ContainerList(context.Background(), container.ListOptions{All: true})
		return containerListMsg(containers)
	}
}
