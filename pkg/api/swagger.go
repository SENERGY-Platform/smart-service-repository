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
	_ "github.com/SENERGY-Platform/smart-service-repository/docs"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/julienschmidt/httprouter"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/swaggo/swag"
	"net/http"
)

func init() {
	endpoints = append(endpoints, &SwaggerEndpoints{})
}

type SwaggerEndpoints struct{}

func (this *SwaggerEndpoints) Swagger(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	if config.EnableSwaggerUi {
		router.GET("/swagger/:any", func(res http.ResponseWriter, req *http.Request, p httprouter.Params) {
			httpSwagger.WrapHandler(res, req)
		})
	}

	router.GET("/doc", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		writer.Header().Set("Content-Type", "application/json; charset=utf-8")
		doc, err := swag.ReadDoc()
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		//remove empty host field
		documentMap := map[string]interface{}{}
		err = json.Unmarshal([]byte(doc), &documentMap)
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if host, ok := documentMap["host"]; ok && host == "" {
			delete(documentMap, "host")
		}
		output, err := json.Marshal(documentMap)
		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		_, _ = writer.Write(output)
	})
}
