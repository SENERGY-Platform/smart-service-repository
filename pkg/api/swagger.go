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
	_ "github.com/SENERGY-Platform/smart-service-repository/docs"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/julienschmidt/httprouter"
	httpSwagger "github.com/swaggo/http-swagger"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &SwaggerEndpoints{})
}

type SwaggerEndpoints struct{}

func (this *SwaggerEndpoints) Swagger(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	if config.UseSwaggerEndpoints {
		router.GET("/swagger/:any", func(res http.ResponseWriter, req *http.Request, p httprouter.Params) {
			httpSwagger.WrapHandler(res, req)
		})
	}
}
