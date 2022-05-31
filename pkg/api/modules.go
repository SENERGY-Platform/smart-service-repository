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
	"net/http"
	"strconv"
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
// @Param        limit query integer false "limits size of result; 0 means unlimited"
// @Param        offset query integer false "offset to be used in combination with limit"
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

		query := model.ModuleQueryOptions{}
		moduleTypeFilter := request.URL.Query().Get("module_type")
		if moduleTypeFilter != "" {
			query.TypeFilter = &moduleTypeFilter
		}
		instanceIdFilter := request.URL.Query().Get("instance_id")
		if instanceIdFilter != "" {
			query.InstanceIdFilter = &instanceIdFilter
		}
		limit := request.URL.Query().Get("limit")
		if limit != "" {
			query.Limit, err = strconv.Atoi(limit)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		offset := request.URL.Query().Get("offset")
		if offset != "" {
			query.Offset, err = strconv.Atoi(offset)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}
		query.Sort = request.URL.Query().Get("sort")
		if query.Sort == "" {
			query.Sort = "name.asc"
		}

		result, err, code := ctrl.ListModules(token, query)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		json.NewEncoder(writer).Encode(result)
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
// @Failure      400
// @Failure      401
// @Router       /modules [post]
func (this *Modules) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/modules", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		module := model.SmartServiceModule{}
		err = json.NewDecoder(request.Body).Decode(&module)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.AddModule(token, module)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		json.NewEncoder(writer).Encode(result)
	})
}
