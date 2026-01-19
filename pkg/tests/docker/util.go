package docker

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

func inCIEnv() bool {
	ci := os.Getenv("CI") == "true"
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
