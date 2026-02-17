package docker

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/testcontainers/testcontainers-go"
)

func inCIEnv() bool {
	ci := os.Getenv("CI") == "true" || runtime.GOOS == "linux"
	if !ci {
		log.Println("Not in CI")
		b, err := json.MarshalIndent(os.Environ(), "", "  ")
		if err != nil {
			fmt.Println("error:", err)
		}
		log.Printf("Env:\n%s\n", string(b))
	}
	return ci
}

type LogConsumer struct {
	Prefix string
}

func (this LogConsumer) Accept(l testcontainers.Log) {
	fmt.Print(this.Prefix + string(l.Content))
}
