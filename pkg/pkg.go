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

package pkg

import (
	"context"
	"time"

	devicerepository "github.com/SENERGY-Platform/device-repository/lib/client"
	permclient "github.com/SENERGY-Platform/permissions-v2/pkg/client"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/api"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/camunda"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/controller"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/database/mongo"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/selectables"
)

func Start(ctx context.Context, config configuration.Config) error {
	db, err := mongo.New(config)
	if err != nil {
		return err
	}

	tokenprovider, err := auth.GetCachedTokenProvider(config)
	if err != nil {
		return err
	}

	cmd, err := controller.New(
		ctx,
		config,
		db,
		permclient.New(config.PermissionsV2Url),
		camunda.New(config),
		selectables.New(config),
		tokenprovider,
		devicerepository.NewClient(config.DeviceRepositoryUrl, nil),
	)
	if err != nil {
		return err
	}
	cleanupResult := cmd.Cleanup(false)
	config.GetLogger().Info("cleanup", "result", cleanupResult)
	duration, err := time.ParseDuration(config.CleanupCycle)
	if err != nil {
		config.GetLogger().Error("unable to parse cleanup cycle", "error", err)
	} else {
		ticker := time.NewTicker(duration)
		go func() {
			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					cleanupResult = cmd.Cleanup(false)
					config.GetLogger().Info("cleanup", "result", cleanupResult)
				}
			}
		}()
	}

	return api.Start(ctx, config, cmd)
}
