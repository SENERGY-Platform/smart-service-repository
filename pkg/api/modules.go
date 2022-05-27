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
	"encoding/json"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &Modules{})
}

type Modules struct{}

// List godoc
// @Summary      returns a list of smart-service models
// @Description  returns a list of smart-service models
// @Produce      json
// @Tags         modules
// @Param        module_type query string false "filter by module type"
// @Param        instance_id query string false "filter by instance id"
// @Success      200 {array} model.SmartServiceModule
// @Failure      500
// @Failure      401
// @Router       /modules [get]
func (this *Modules) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/modules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode([]model.SmartServiceModule{})
	})
}

// Create godoc
// @Summary      create a smart-service module
// @Description  creates a smart-service module
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        id path string true "Module ID"
// @Param        message body model.SmartServiceModule true "SmartServiceModule"
// @Success      200 {object} model.SmartServiceModule
// @Failure      500
// @Failure      401
// @Router       /modules [post]
func (this *Modules) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/modules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		//TODO: replace with real code
		log.Println(token)
		json.NewEncoder(writer).Encode(model.SmartServiceModule{})
	})
}
