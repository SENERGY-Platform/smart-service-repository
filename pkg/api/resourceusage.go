/*
 * Copyright 2026 InfAI (CC SES)
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
	"net/http"

	"github.com/SENERGY-Platform/smart-service-repository/pkg/auth"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"github.com/julienschmidt/httprouter"
)

func init() {
	endpoints = append(endpoints, &ResourceUsage{})
}

type ResourceUsage struct{}

// Get godoc
// @Summary      tells which smart-service releases use a resource
// @Description  counts the releases and instances of all users that use the process model, flow or import type with the given id; a release counts if it is not deleted and is the newest of its design or has instances; readable lists only the counted releases the caller may read; any authenticated user may ask
// @Tags         releases, resource-usage
// @Produce      json
// @Param        kind path string true "resource kind" Enums(process-models, flows, import-types)
// @Param        id path string true "resource id"
// @Success      200 {object} model.ResourceUsage
// @Failure      400
// @Failure      401
// @Failure      500
// @Router       /resource-usage/{kind}/{id} [get]
func (this *ResourceUsage) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/resource-usage/:kind/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		result, err, code := ctrl.GetResourceUsage(request.Context(), token, model.ResourceKind(params.ByName("kind")), params.ByName("id"))
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}
