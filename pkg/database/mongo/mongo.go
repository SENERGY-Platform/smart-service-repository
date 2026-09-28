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
	"errors"
	"fmt"
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

var (
	ErrEmptyDatabase   = errors.New("mongo database name must not be empty")
	ErrMissingPassword = errors.New("mongo password must not be empty when a mongo user is set")
)

const startupCheckTimeout = 10 * time.Second

func New(conf configuration.Config) (*Mongo, error) {
	if err := validateConfig(conf); err != nil {
		return nil, err
	}
	ctx, _ := getTimeoutContext()
	//otelmongo needs an initialized open-telemetry; mongo.New() may run before the api is started.
	//the returned handler is unused, the call is done for the initialisation, which happens only once per process.
	_, err := otelx.GinOpenTelemetry(context.Background(), configuration.ServiceName, conf.OtelEndpoint)
	if err != nil {
		conf.GetLogger().Error("unable to init open-telemetry -> continue without tracing", "error", err)
	}
	reg := bson.NewRegistryBuilder().RegisterTypeMapEntry(bsontype.EmbeddedDocument, reflect.TypeOf(bson.M{})).Build() //ensure map marshalling to interface
	opts := clientOptions(conf).SetRegistry(reg).SetMonitor(otelmongo.NewMonitor())
	return start(ctx, conf, opts, startupCheckTimeout)
}

// start disconnects the client on every failure path, so a failed startup leaves nothing connected.
func start(ctx context.Context, conf configuration.Config, opts *options.ClientOptions, timeout time.Duration) (*Mongo, error) {
	client, err := connect(ctx, opts, conf.MongoDatabase, timeout)
	if err != nil {
		return nil, err
	}
	db := &Mongo{config: conf, client: client}
	for _, creators := range CreateCollections {
		err = creators(db)
		if err != nil {
			disconnect(client)
			return nil, err
		}
	}
	return db, nil
}

// connect runs listCollections on the service database because Connect is lazy and ping needs no
// authentication; unreachable servers and wrong or missing credentials then fail at startup.
func connect(ctx context.Context, opts *options.ClientOptions, database string, timeout time.Duration) (*mongo.Client, error) {
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, err
	}
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	listOpts := options.ListCollections().SetNameOnly(true).SetAuthorizedCollections(true)
	if _, err = client.Database(database).ListCollectionNames(checkCtx, bson.D{}, listOpts); err != nil {
		disconnect(client)
		return nil, fmt.Errorf("mongo startup check failed: %w", err)
	}
	return client, nil
}

// disconnect uses a fresh context, because the caller's context may be the reason for the failure.
func disconnect(client *mongo.Client) {
	ctx, cancel := getTimeoutContext()
	defer cancel()
	_ = client.Disconnect(ctx)
}

func validateConfig(conf configuration.Config) error {
	if conf.MongoDatabase == "" {
		return ErrEmptyDatabase
	}
	if conf.MongoUser != "" && conf.MongoPassword == "" {
		return ErrMissingPassword
	}
	return nil
}

// clientOptions applies the credentials after the URI so they replace any given in MONGO_URL.
func clientOptions(conf configuration.Config) *options.ClientOptions {
	opts := options.Client().ApplyURI(conf.MongoUrl)
	if conf.MongoUser != "" {
		opts.SetAuth(options.Credential{
			Username:   conf.MongoUser,
			Password:   conf.MongoPassword,
			AuthSource: conf.MongoAuthSource,
		})
	}
	return opts
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
