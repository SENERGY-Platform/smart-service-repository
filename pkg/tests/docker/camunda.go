package docker

import (
	"context"
	"fmt"
	"github.com/SENERGY-Platform/permission-search/lib/tests/docker"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"sync"
)

func Camunda(ctx context.Context, wg *sync.WaitGroup, pgIp string, pgPort string) (camundaUrl string, err error) {
	log.Println("start camunda")
	dbName := "camunda"
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "ghcr.io/senergy-platform/process-engine:dev",
			ExposedPorts: []string{"8080/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("8080/tcp"),
				wait.ForLog("Server startup in"),
			),
			Env: map[string]string{
				"DB_PASSWORD": "pw",
				"DB_URL":      "jdbc:postgresql://" + pgIp + ":" + pgPort + "/" + dbName,
				"DB_PORT":     pgPort,
				"DB_NAME":     dbName,
				"DB_HOST":     pgIp,
				"DB_DRIVER":   "org.postgresql.Driver",
				"DB_USERNAME": "usr",
				"DATABASE":    "postgres",
			},
		},
		Started: true,
	})
	if err != nil {
		return "", err
	}

	err = docker.Dockerlog(ctx, c, "CAMUNDA")
	if err != nil {
		return "", err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container camunda", c.Terminate(context.Background()))
	}()

	containerip, err := c.ContainerIP(ctx)
	if err != nil {
		return "", err
	}

	camundaUrl = fmt.Sprintf("http://%s:%s", containerip, "8080")

	return camundaUrl, err
}
