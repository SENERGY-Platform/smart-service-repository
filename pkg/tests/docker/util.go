package docker

import (
	"os"
)

func inCIEnv() bool {
	os.Setenv("CI", "false")
	return os.Getenv("CI") == "true"
}
