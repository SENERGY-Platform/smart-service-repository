/*
 * Copyright 2019 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kafka

import (
	"context"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/tests/docker"
	"github.com/ory/dockertest/v3"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestKafka(t *testing.T) {
	if testing.Short() {
		t.Skip("short tests only without docker")
	}
	config, err := configuration.Load("../../config.json")
	if err != nil {
		t.Error(err)
		return
	}
	config.Debug = true

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Error(err)
		return
	}

	closeZk, _, zkIp, err := docker.Zookeeper(pool)
	if err != nil {
		t.Error(err)
		return
	}
	defer closeZk()
	zkUrl := zkIp + ":2181"

	//kafka
	var closeKafka func()
	config.KafkaUrl, closeKafka, err = docker.Kafka(pool, zkUrl)
	if err != nil {
		t.Error(err)
		return
	}
	defer closeKafka()

	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumed := []string{}
	mux := sync.Mutex{}

	wait := sync.WaitGroup{}

	err = NewConsumer(ctx, config, "test", func(delivery []byte) error {
		mux.Lock()
		defer mux.Unlock()
		consumed = append(consumed, string(delivery))
		wait.Done()
		return nil
	})

	if err != nil {
		t.Error(err)
		return
	}

	producer, err := NewProducer(ctx, config, "test")
	if err != nil {
		t.Error(err)
		return
	}

	wait.Add(1)
	err = producer.Produce("key", []byte("foo"))
	if err != nil {
		t.Error(err)
		return
	}

	wait.Add(1)
	err = producer.Produce("key", []byte("bar"))
	if err != nil {
		t.Error(err)
		return
	}

	wait.Wait()
	mux.Lock()
	defer mux.Unlock()

	if !reflect.DeepEqual(consumed, []string{"foo", "bar"}) {
		t.Error(consumed)
		return
	}
	//wait for finished commits
	time.Sleep(1 * time.Second)
}
