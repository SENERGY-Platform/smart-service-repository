package docker

import (
	"os"
)

func inCIEnv() bool {
	os.Getenv("CI")
	return os.Getenv("CI") == "true"
}
