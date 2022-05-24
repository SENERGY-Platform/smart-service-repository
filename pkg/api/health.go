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

package api

import (
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/julienschmidt/httprouter"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &HealthEndpoints{})
}

type HealthEndpoints struct{}

// HealthCheck godoc
// @Summary      health check
// @Description  checks health and reachability of the service
// @Tags         health
// @Success      200
// @Router       / [get]
func (this *HealthEndpoints) HealthCheck(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		writer.WriteHeader(200)
	})
}

// HealthCheck godoc
// @Summary      health check
// @Description  checks health and reachability of the service
// @Tags         health
// @Success      200
// @Router       /health [get]
func (this *HealthEndpoints) HealthCheck2(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/health", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		writer.WriteHeader(200)
	})
}
