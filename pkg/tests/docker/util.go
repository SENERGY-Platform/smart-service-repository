package docker

import (
	"fmt"
	"os"

	"github.com/testcontainers/testcontainers-go"
)

func inCIEnv() bool {
	return os.Getenv("WSL_DISTRO_NAME") == ""
}

type LogConsumer struct {
	Prefix string
}

func (this LogConsumer) Accept(l testcontainers.Log) {
	fmt.Print(this.Prefix + string(l.Content))
}
