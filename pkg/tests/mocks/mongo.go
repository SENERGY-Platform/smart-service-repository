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
	"github.com/strikesecurity/strikememongo"
	"sync"
)

func Mongo(ctx context.Context, wg *sync.WaitGroup) (mongoUrl string, err error) {
	mongoServer, err := strikememongo.StartWithOptions(&strikememongo.Options{MongoVersion: "4.2.1", ShouldUseReplica: true})
	if err != nil {
		return "", err
	}
	wg.Add(1)
	go func() {
		<-ctx.Done()
		mongoServer.Stop()
		wg.Done()
	}()

	return mongoServer.URI(), nil
}
