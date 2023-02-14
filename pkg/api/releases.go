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
	"sync"
)

func init() {
	endpoints = append(endpoints, &Releases{})
}

type Releases struct{}

// Create godoc
// @Summary      create a smart-service release
// @Description  creates a smart-service release
// @Tags         releases
// @Accept       json
// @Produce      json
// @Param        id path string true "Release ID"
// @Param        message body model.SmartServiceRelease true "SmartServiceRelease"
// @Success      200 {object} model.SmartServiceRelease
// @Failure      500
// @Failure      401
// @Router       /releases [post]
func (this *Releases) Create(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/releases", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		element := model.SmartServiceRelease{}
		err = json.NewDecoder(request.Body).Decode(&element)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		if element.Id == "" {
			element.Id = ctrl.GetNewId()
		}

		result, err, code := ctrl.CreateRelease(token, element)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// Delete godoc
// @Summary      removes a smart-service release
// @Description  removes a smart-service release
// @Tags         releases
// @Accept       json
// @Produce      json
// @Param        id path string true "Release ID"
// @Success      200
// @Failure      500
// @Failure      400
// @Failure      401
// @Failure      403
// @Router       /releases/{id} [delete]
func (this *Releases) Delete(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.DELETE("/releases/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing id", http.StatusBadRequest)
			return
		}

		err, code := ctrl.DeleteRelease(token, id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
}

// Get godoc
// @Summary      returns a smart-service release
// @Description  returns a smart-service release
// @Tags         releases
// @Produce      json
// @Param        id path string true "Release ID"
// @Success      200 {object} model.SmartServiceRelease
// @Failure      500
// @Failure      401
// @Router       /releases/{id} [get]
func (this *Releases) Get(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/releases/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing id", http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.GetRelease(token, id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// List godoc
// @Summary      returns a list of smart-service releases
// @Description  returns a list of smart-service releases
// @Tags         releases
// @Param        limit query integer false "limits size of result"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Param        rights query string false "rights needed to see a release; bay be a combination of the following letters: 'rwxa'; default = r; release rights are set with https://github.com/SENERGY-Platform/permission-command"
// @Param        sort query string false "describes the sorting in the form of name.asc"
// @Param		 search query string false "optional text search (permission-search/elastic-search behavior)"
// @Param        latest query bool false "returns only newest release of the same design"
// @Param        add-usable-flag query bool false "add 'usable' flag to result, describing if the user hase options for all iot parameters"
// @Produce      json
// @Success      200 {array} model.SmartServiceRelease
// @Failure      500
// @Failure      401
// @Router       /releases [get]
func (this *Releases) List(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/releases", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		query := model.ReleaseQueryOptions{
			Limit: 100,
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
		query.Rights = request.URL.Query().Get("rights")
		if query.Rights == "" {
			query.Rights = "r"
		}
		query.Sort = request.URL.Query().Get("sort")
		if query.Sort == "" {
			query.Sort = "name.asc"
		}
		query.Search = request.URL.Query().Get("search")

		latestStr := request.URL.Query().Get("latest")
		if latestStr != "" {
			query.Latest, err = strconv.ParseBool(latestStr)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}

		addUsableFlagStr := request.URL.Query().Get("add-usable-flag")
		addUsableFlag := false
		if addUsableFlagStr != "" {
			addUsableFlag, err = strconv.ParseBool(addUsableFlagStr)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}

		result, err, code := ctrl.ListReleases(token, query)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}

		if addUsableFlag {
			withUsableFlat, err := addUsableFlagToReleases(ctrl, token, result)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			json.NewEncoder(writer).Encode(withUsableFlat)
		} else {
			writer.Header().Set("Content-Type", "application/json")
			json.NewEncoder(writer).Encode(result)
		}
	})
}

// GetExtended godoc
// @Summary      returns a smart-service release
// @Description  returns a smart-service release
// @Tags         releases
// @Produce      json
// @Param        id path string true "Release ID"
// @Success      200 {object} model.SmartServiceReleaseExtended
// @Failure      500
// @Failure      401
// @Router       /extended-releases/{id} [get]
func (this *Releases) GetExtended(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/extended-releases/:id", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing id", http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.GetExtendedRelease(token, id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// ListExtended godoc
// @Summary      returns a list of smart-service releases
// @Description  returns a list of smart-service releases
// @Tags         releases
// @Param        limit query integer false "limits size of result"
// @Param        offset query integer false "offset to be used in combination with limit"
// @Param        rights query string false "rights needed to see a release; bay be a combination of the following letters: 'rwxa'; default = r; release rights are set with https://github.com/SENERGY-Platform/permission-command"
// @Param        sort query string false "describes the sorting in the form of name.asc"
// @Param		 search query string false "optional text search (permission-search/elastic-search behavior)"
// @Param        latest query bool false "returns only newest release of the same design"
// @Param        add-usable-flag query bool false "add 'usable' flag to result, describing if the user hase options for all iot parameters"
// @Produce      json
// @Success      200 {array} model.SmartServiceReleaseExtended
// @Failure      500
// @Failure      401
// @Router       /extended-releases [get]
func (this *Releases) ListExtended(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/extended-releases", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		query := model.ReleaseQueryOptions{
			Limit: 100,
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
		query.Rights = request.URL.Query().Get("rights")
		if query.Rights == "" {
			query.Rights = "r"
		}
		query.Sort = request.URL.Query().Get("sort")
		if query.Sort == "" {
			query.Sort = "name.asc"
		}
		query.Search = request.URL.Query().Get("search")

		latestStr := request.URL.Query().Get("latest")
		if latestStr != "" {
			query.Latest, err = strconv.ParseBool(latestStr)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}

		addUsableFlagStr := request.URL.Query().Get("add-usable-flag")
		addUsableFlag := false
		if addUsableFlagStr != "" {
			addUsableFlag, err = strconv.ParseBool(addUsableFlagStr)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusBadRequest)
				return
			}
		}

		result, err, code := ctrl.ListExtendedReleases(token, query)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}

		if addUsableFlag {
			withUsableFlat, err := addUsableFlagToExtendedReleases(ctrl, token, result)
			if err != nil {
				http.Error(writer, err.Error(), code)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			json.NewEncoder(writer).Encode(withUsableFlat)
		} else {
			writer.Header().Set("Content-Type", "application/json")
			json.NewEncoder(writer).Encode(result)
		}
	})
}

func addUsableFlagToExtendedReleases(ctrl Controller, token auth.Token, releases []model.SmartServiceReleaseExtended) (result []model.SmartServiceReleaseExtendedWithUsableFlag, err error) {
	wg := sync.WaitGroup{}
	mux := sync.Mutex{}
	for _, release := range releases {
		wg.Add(1)
		go func(release model.SmartServiceReleaseExtended) {
			defer wg.Done()
			parameter, temperr, _ := ctrl.GetReleaseParameterWithoutAuthCheck(token, release.Id)
			if temperr != nil {
				err = temperr
				return
			}
			element := model.SmartServiceReleaseExtendedWithUsableFlag{
				SmartServiceReleaseExtended: release,
				Usable:                      true,
			}
			for _, param := range parameter {
				if param.HasNoValidOption {
					element.Usable = false
				}
			}
			mux.Lock()
			defer mux.Unlock()
			result = append(result, element)
		}(release)
	}
	wg.Wait()
	return result, err
}

func addUsableFlagToReleases(ctrl Controller, token auth.Token, releases []model.SmartServiceRelease) (result []model.SmartServiceReleaseWithUsableFlag, err error) {
	for _, release := range releases {
		parameter, err, _ := ctrl.GetReleaseParameterWithoutAuthCheck(token, release.Id)
		if err != nil {
			return result, err
		}
		element := model.SmartServiceReleaseWithUsableFlag{
			SmartServiceRelease: release,
			Usable:              true,
		}
		for _, param := range parameter {
			if param.HasNoValidOption {
				element.Usable = false
			}
		}
		result = append(result, element)
	}
	return result, nil
}

// Parameters godoc
// @Summary      returns parameters of a release
// @Description  returns parameters of a release
// @Tags         releases, parameter
// @Produce      json
// @Param        id path string true "Release ID"
// @Success      200 {array} model.SmartServiceExtendedParameter
// @Failure      500
// @Failure      401
// @Router       /releases/{id}/parameters [get]
func (this *Releases) Parameters(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.GET("/releases/:id/parameters", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}

		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing id", http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.GetReleaseParameter(token, id)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}

// Start godoc
// @Summary      creates a smart-service instance from the release
// @Description  creates a smart-service instance from the release
// @Tags         releases, instances
// @Accept       json
// @Produce      json
// @Param        id path string true "Release ID"
// @Param        message body model.SmartServiceInstanceInit true "SmartServiceInstanceInit"
// @Success      200 {object} model.SmartServiceInstance
// @Failure      500
// @Failure      401
// @Router       /releases/{id}/instances [post]
func (this *Releases) Start(config configuration.Config, router *httprouter.Router, ctrl Controller) {
	router.POST("/releases/:id/instances", func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		token, err := auth.GetParsedToken(request)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusUnauthorized)
			return
		}
		id := params.ByName("id")
		if id == "" {
			http.Error(writer, "missing release id", http.StatusBadRequest)
			return
		}
		instance := model.SmartServiceInstanceInit{}
		err = json.NewDecoder(request.Body).Decode(&instance)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		result, err, code := ctrl.CreateInstance(token, id, instance)
		if err != nil {
			http.Error(writer, err.Error(), code)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		json.NewEncoder(writer).Encode(result)
	})
}
