package docker

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"log"
	"sync"
)

func Postgres(ctx context.Context, wg *sync.WaitGroup, dbname string) (conStr string, ip string, port string, err error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return "", ip, port, err
	}
	container, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11.2",
		Env:        []string{"POSTGRES_DB=" + dbname, "POSTGRES_PASSWORD=pw", "POSTGRES_USER=usr"},
	}, func(config *docker.HostConfig) {
		config.Tmpfs = map[string]string{"/var/lib/postgresql/data": "rw"}
	})
	if err != nil {
		return "", ip, port, err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		log.Println("DEBUG: remove container " + container.Container.Name)
		container.Close()
		wg.Done()
	}()
	ip = container.Container.NetworkSettings.IPAddress
	port = "5432"
	conStr = fmt.Sprintf("postgres://usr:pw@%s:%s/%s?sslmode=disable", ip, port, dbname)
	err = pool.Retry(func() error {
		var err error
		log.Println("try connecting to pg")
		db, err := sql.Open("postgres", conStr)
		if err != nil {
			log.Println(err)
			return err
		}
		err = db.Ping()
		if err != nil {
			log.Println(err)
			return err
		}
		return nil
	})
	return
}
