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

package configuration

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerPort                           string   `json:"server_port"`
	Debug                                bool     `json:"debug"`
	EnableSwaggerUi                      bool     `json:"enable_swagger_ui"`
	CamundaUrl                           string   `json:"camunda_url" config:"secret"`
	DeviceSelectionApi                   string   `json:"device_selection_api"`
	PermissionsV2Url                     string   `json:"permissions_v2_url"`
	DeviceRepositoryUrl                  string   `json:"device_repository_url"`
	SmartServiceReleasePermissionsTopic  string   `json:"smart_service_release_permissions_topic"`
	NotificationUrl                      string   `json:"notification_url"`
	MongoUrl                             string   `json:"mongo_url"`
	MongoWithTransactions                bool     `json:"mongo_with_transactions"`
	MongoTable                           string   `json:"mongo_table"`
	MongoCollectionDesign                string   `json:"mongo_collection_design"`
	MongoCollectionRelease               string   `json:"mongo_collection_release"`
	MongoCollectionInstance              string   `json:"mongo_collection_instance"`
	MongoCollectionModule                string   `json:"mongo_collection_module"`
	MongoCollectionVariables             string   `json:"mongo_collection_variables"`
	AuthEndpoint                         string   `json:"auth_endpoint"`
	AuthClientId                         string   `json:"auth_client_id" config:"secret"`
	AuthClientSecret                     string   `json:"auth_client_secret" config:"secret"`
	AuthExpirationTimeBuffer             float64  `json:"auth_expiration_time_buffer"`
	TokenCacheDefaultExpirationInSeconds int      `json:"token_cache_default_expiration_in_seconds"`
	CleanupCycle                         string   `json:"cleanup_cycle"`
	MarkAgeLimit                         Duration `json:"mark_age_limit"`
}

// loads config from json in location and used environment variables (e.g KafkaUrl --> KAFKA_URL)
func Load(location string) (config Config, err error) {
	file, err := os.Open(location)
	if err != nil {
		return config, err
	}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return config, err
	}
	err = handleEnvironmentVars(&config)
	if err != nil {
		return config, err
	}
	return config, nil
}

var camel = regexp.MustCompile("(^[^A-Z]*|[A-Z]*)([A-Z][^A-Z]+|$)")

func fieldNameToEnvName(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToUpper(strings.Join(a, "_"))
}

// preparations for docker
func handleEnvironmentVars(config *Config) (err error) {
	configValue := reflect.Indirect(reflect.ValueOf(config))
	configType := configValue.Type()
	for index := 0; index < configType.NumField(); index++ {
		fieldName := configType.Field(index).Name
		fieldConfig := configType.Field(index).Tag.Get("config")
		envName := fieldNameToEnvName(fieldName)
		envValue := os.Getenv(envName)
		if envValue != "" {
			loggedEnvValue := envValue
			if strings.Contains(fieldConfig, "secret") {
				loggedEnvValue = "***"
			}
			fmt.Println("use environment variable: ", envName, " = ", loggedEnvValue)
			if field := configValue.FieldByName(fieldName); field.Kind() == reflect.Struct && field.CanInterface() {
				fieldPtrInterface := field.Addr().Interface()
				setter, setterOk := fieldPtrInterface.(interface{ SetString(string) })
				errSetter, errSetterOk := fieldPtrInterface.(interface{ SetString(string) error })
				if setterOk {
					setter.SetString(envValue)
				}
				if errSetterOk {
					err = errSetter.SetString(envValue)
					if err != nil {
						return fmt.Errorf("invalid env variable %v=%v: %w", envName, envValue, err)
					}
				}
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Int64 || configValue.FieldByName(fieldName).Kind() == reflect.Int {
				i, _ := strconv.ParseInt(envValue, 10, 64)
				configValue.FieldByName(fieldName).SetInt(i)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.String {
				configValue.FieldByName(fieldName).SetString(envValue)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Bool {
				b, _ := strconv.ParseBool(envValue)
				configValue.FieldByName(fieldName).SetBool(b)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Float64 {
				f, _ := strconv.ParseFloat(envValue, 64)
				configValue.FieldByName(fieldName).SetFloat(f)
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Slice {
				val := []string{}
				for _, element := range strings.Split(envValue, ",") {
					val = append(val, strings.TrimSpace(element))
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(val))
			}
			if configValue.FieldByName(fieldName).Kind() == reflect.Map {
				value := map[string]string{}
				for _, element := range strings.Split(envValue, ",") {
					keyVal := strings.Split(element, ":")
					key := strings.TrimSpace(keyVal[0])
					val := strings.TrimSpace(keyVal[1])
					value[key] = val
				}
				configValue.FieldByName(fieldName).Set(reflect.ValueOf(value))
			}
		}
	}
	return nil
}

type Duration struct {
	dur time.Duration
}

func (this *Duration) GetDuration() time.Duration {
	return this.dur
}

func (this *Duration) SetDuration(dur time.Duration) {
	this.dur = dur
}

func (this *Duration) SetString(str string) error {
	if str == "" {
		return nil
	}
	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	this.SetDuration(duration)
	return nil
}

func (this *Duration) UnmarshalJSON(bytes []byte) (err error) {
	var str string
	err = json.Unmarshal(bytes, &str)
	if err != nil {
		return err
	}
	return this.SetString(str)
}
