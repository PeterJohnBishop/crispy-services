package manager

import (
	"time"

	"github.com/docker/docker/api/types/container"
)

type containerListMsg []container.Summary
type logLineMsg string
type tickMsg time.Time
