/*
 * Copyright (c) 2022 InfAI (CC SES)
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

package mocks

import (
	"context"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
)

type ProducerMock struct {
	Receiver func(topic string, key string, message []byte) error
	Topic    string
}

func (this *ProducerMock) Produce(key string, message []byte) error {
	return this.Receiver(this.Topic, key, message)
}

func NewProducer(receiver func(topic string, key string, message []byte) error) controller.ProducerFactory {
	return controller.NewProducerFactory(func(ctx context.Context, config configuration.Config, topic string) (*ProducerMock, error) {
		return &ProducerMock{Receiver: receiver, Topic: topic}, nil
	})
}

func NewConsumer(errHandler func(err error)) (sender func(topic string, message []byte), consumer controller.Consumer) {
	listeners := map[string]func(delivery []byte) error{}
	consumer = func(ctx context.Context, config configuration.Config, topic string, listener func(delivery []byte) error) error {
		listeners[topic] = listener
		return nil
	}
	sender = func(topic string, message []byte) {
		l, ok := listeners[topic]
		if ok {
			err := l(message)
			if err != nil && errHandler != nil {
				errHandler(err)
			}
		}
	}
	return sender, consumer
}
