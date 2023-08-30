package docker

import (
	"context"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"log"
	"sync"
)

func Postgres(ctx context.Context, wg *sync.WaitGroup, dbname string) (conStr string, ip string, port string, err error) {
	log.Println("start postgres")
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "postgres:11.2",
			Env: map[string]string{
				"POSTGRES_DB":       dbname,
				"POSTGRES_PASSWORD": "pw",
				"POSTGRES_USER":     "usr",
			},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5432/tcp"),
			),
			Tmpfs: map[string]string{"/var/lib/postgresql/data": "rw"},
		},
		Started: true,
	})
	if err != nil {
		return "", "", "", err
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Println("DEBUG: remove container postgres", c.Terminate(context.Background()))
	}()

	ip, err = c.ContainerIP(ctx)
	if err != nil {
		return "", "", "", err
	}
	temp, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return "", "", "", err
	}
	port = temp.Port()
	conStr = fmt.Sprintf("postgres://usr:pw@%s:%s/%s?sslmode=disable", ip, "5432", dbname)

	return conStr, ip, port, err
}
