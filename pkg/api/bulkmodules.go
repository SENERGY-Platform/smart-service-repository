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
)

func init() {
	endpoints = append(endpoints, &BulkModules{})
}

type BulkModules struct{}

// CreateBulkByProcessInstance godoc
// @Summary      create smart-service modules
// @Description  creates smart-service modules; only usable if config.json mongo_url points to a mongodb capable of transactions (replication-set)
// @Tags         modules
// @Accept       json
// @Produce      json
// @Param        id path string true "Process-Instance ID"
// @Param        message body model.SmartServiceModuleInitList true "list of SmartServiceModuleInit"
// @Success      200 {array} model.SmartServiceModule
// @Failure      500
// @Failure      400
// @Failure      401
// @Router       /instances-by-process-id/{id}/modules/bulk [post]
func (this *BulkModules) CreateBulkByProcessInstance(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/instances-by-process-id/:id/modules/bulk", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		if !token.IsAdmin() {
			http.Error(writer, "only admins may ask for instance user-id", http.StatusForbidden)
			return
		}

		modules := []model.SmartServiceModuleInit{}
		err = json.NewDecoder(request.Body).Decode(&modules)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.AddModulesForProcessInstance(params.ByName("id"), modules)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}
