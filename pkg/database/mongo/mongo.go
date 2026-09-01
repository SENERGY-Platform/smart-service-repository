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

package mongo

import (
	"context"
	"reflect"
	"time"

	"github.com/SENERGY-Platform/gin-middleware/otelx"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.opentelemetry.io/contrib/instrumentation/go.mongodb.org/mongo-driver/mongo/otelmongo"
)

type Mongo struct {
	config configuration.Config
	client *mongo.Client
}

var CreateCollections = []func(db *Mongo) error{}

func New(conf configuration.Config) (*Mongo, error) {
	ctx, _ := getTimeoutContext()
	//otelmongo needs an initialized open-telemetry; mongo.New() may run before the api is started.
	//the returned handler is unused, the call is done for the initialisation, which happens only once per process.
	_, err := otelx.GinOpenTelemetry(context.Background(), configuration.ServiceName, conf.OtelEndpoint)
	if err != nil {
		conf.GetLogger().Error("unable to init open-telemetry -> continue without tracing", "error", err)
	}
	reg := bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build() //ensure map marshalling to interface
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(conf.MongoUrl), options.Client().SetRegistry(reg), options.Client().SetMonitor(otelmongo.NewMonitor()))
	if err != nil {
		return nil, err
	}
	db := &Mongo{config: conf, client: client}
	for _, creators := range CreateCollections {
		err = creators(db)
		if err != nil {
			client.Disconnect(context.Background())
			return nil, err
		}
	}
	return db, nil
}

// getTimeoutContext derives a context with the mongo timeout from the given parent.
// context.WithoutCancel keeps the trace-context and the baggage of the parent, which is
// what the otelmongo monitor and the logging need, but not its cancellation: a client that
// aborts its request must not cancel a write that is already running.
func getTimeoutContext(parent ...context.Context) (context.Context, context.CancelFunc) {
	if len(parent) > 0 && parent[0] != nil {
		return context.WithTimeout(context.WithoutCancel(parent[0]), 10*time.Second)
	}
	return context.WithTimeout(context.Background(), 10*time.Second)
}
