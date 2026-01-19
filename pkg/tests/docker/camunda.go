package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func Camunda(ctx context.Context, wg *sync.WaitGroup, pgPort string, conStr string) (camundaUrl string, err error) {
	log.Println("start camunda")
	dbName := "camunda"
	env := map[string]string{
		"DB_PASSWORD": "pw",
		"DB_URL":      "jdbc:postgresql://host.docker.internal:" + pgPort + "/" + dbName,
		"DB_PORT":     pgPort,
		"DB_NAME":     dbName,
		"DB_HOST":     "host.docker.internal",
		"DB_DRIVER":   "org.postgresql.Driver",
		"DB_USERNAME": "usr",
		"DATABASE":    "postgres",
	}
	b, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		fmt.Println("error:", err)
	}
	log.Printf("Env:\n%s\n", string(b))

	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/senergy-platform/process-engine:v1.0.2", // dev | v1.0.2 | v1.0.4
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("8080/tcp"),
				wait.ForLog("Server startup in"),
			),
			Env: env,
		},
		Started: true,
	})
	if err != nil {
		return "", err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container camunda", c.Terminate(context.Background()))
	}()

	temp, err := c.MappedPort(ctx, "8080/tcp")
	if err != nil {
		return "", err
	}
	hostport := temp.Port()

	camundaUrl = fmt.Sprintf("http://%s:%s", "127.0.0.1", hostport)

	return camundaUrl, err
}
