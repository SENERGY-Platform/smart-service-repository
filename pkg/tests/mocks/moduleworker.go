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

package mocks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/configuration"
	"github.com/SENERGY-Platform/smart-service-repository/pkg/model"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"sync"
	"time"
)

var CAMUNDA_MODULE_WORKER_TOPIC = "process_deployment"
var MOCK_MODULE_DELETE_INFO *model.ModuleDeleteInfo = nil
var WORKER_ID = "test"

type ModuleWorkerMessage = CamundaExternalTask

func NewModuleWorker(ctx context.Context, wg *sync.WaitGroup, smartServiceRepoApi string, config configuration.Config, handler func(taskWorkerMsg ModuleWorkerMessage) (err error)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				wait := executeNextTasks(smartServiceRepoApi, config, handler)
				if wait {
					duration := 200 * time.Millisecond
					time.Sleep(duration)
				}
			}
		}
	}()
}

func executeNextTasks(smartServiceRepoApi string, config configuration.Config, handler func(taskWorkerMsg ModuleWorkerMessage) (err error)) (wait bool) {
	tasks, err := getTasks(config)
	if err != nil {
		log.Println("error on ExecuteNextTasks getTask", err)
		return true
	}
	if len(tasks) == 0 {
		return true
	}
	for _, task := range tasks {
		err = handler(task)
		if err != nil {
			sendWorkerError(smartServiceRepoApi, task, err)
			err = completeTask(config.CamundaUrl, task.Id)
			if err != nil {
				log.Println("ERROR", err)
				debug.PrintStack()
			}
		} else {
			sendWorkerModule(smartServiceRepoApi, task)
			err = completeTask(config.CamundaUrl, task.Id)
			if err != nil {
				log.Println("ERROR", err)
				debug.PrintStack()
			}
		}
	}
	return false
}

func getTasks(config configuration.Config) (tasks []CamundaExternalTask, err error) {
	fetchRequest := CamundaFetchRequest{
		WorkerId: WORKER_ID,
		MaxTasks: 100,
		Topics:   []CamundaTopic{{LockDuration: 60000, Name: CAMUNDA_MODULE_WORKER_TOPIC}},
	}
	client := http.Client{Timeout: 5 * time.Second}
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(fetchRequest)
	if err != nil {
		return
	}
	endpoint := config.CamundaUrl + "/engine-rest/external-task/fetchAndLock"
	resp, err := client.Post(endpoint, "application/json", b)
	if err != nil {
		return tasks, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		temp, err := ioutil.ReadAll(resp.Body)
		err = errors.New(fmt.Sprintln(endpoint, resp.Status, resp.StatusCode, string(temp), err))
		return tasks, err
	}
	err = json.NewDecoder(resp.Body).Decode(&tasks)
	return
}

type CamundaFetchRequest struct {
	WorkerId string         `json:"workerId,omitempty"`
	MaxTasks int64          `json:"maxTasks,omitempty"`
	Topics   []CamundaTopic `json:"topics,omitempty"`
}

type CamundaTopic struct {
	Name         string `json:"topicName,omitempty"`
	LockDuration int64  `json:"lockDuration,omitempty"`
}

type CamundaExternalTask struct {
	Id                  string                     `json:"id,omitempty"`
	Variables           map[string]CamundaVariable `json:"variables,omitempty"`
	ActivityId          string                     `json:"activityId,omitempty"`
	Retries             int64                      `json:"retries"`
	ExecutionId         string                     `json:"executionId"`
	ProcessInstanceId   string                     `json:"processInstanceId"`
	ProcessDefinitionId string                     `json:"processDefinitionId"`
	TenantId            string                     `json:"tenantId"`
	Error               string                     `json:"errorMessage"`
}

type CamundaVariable struct {
	Type  string      `json:"type,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

func sendWorkerModule(api string, task CamundaExternalTask) {
	//time.Sleep(2000 * time.Millisecond)
	module := model.SmartServiceModuleInit{
		DeleteInfo: MOCK_MODULE_DELETE_INFO,
		ModuleType: CAMUNDA_MODULE_WORKER_TOPIC,
		ModuleData: map[string]interface{}{
			"camunda-data": task.Variables,
		},
	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(module)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	req, err := http.NewRequest("POST", api+"/instances-by-process-id/"+url.PathEscape(task.ProcessInstanceId)+"/modules", body)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		log.Println("ERROR:", resp.StatusCode, string(temp))
		debug.PrintStack()
		return
	}
	return
}

func sendWorkerError(api string, task CamundaExternalTask, err error) {
	body := new(bytes.Buffer)
	err = json.NewEncoder(body).Encode(err.Error())
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	req, err := http.NewRequest("PUT", api+"/instances-by-process-id/"+url.PathEscape(task.ProcessInstanceId)+"/error", body)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	req.Header.Set("Authorization", adminToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("ERROR:", err)
		debug.PrintStack()
		return
	}
	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		log.Println("ERROR:", resp.StatusCode, string(temp))
		debug.PrintStack()
		return
	}
	return
}

func completeTask(api string, taskId string) (err error) {
	log.Println("Start complete Request")
	client := http.Client{Timeout: 5 * time.Second}

	var completeRequest = CamundaCompleteRequest{WorkerId: WORKER_ID}
	b := new(bytes.Buffer)
	err = json.NewEncoder(b).Encode(completeRequest)
	if err != nil {
		return
	}
	resp, err := client.Post(api+"/engine-rest/external-task/"+taskId+"/complete", "application/json", b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	pl, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		temp, _ := io.ReadAll(resp.Body)
		log.Println("WARNING: unable to complete task:", resp.StatusCode, string(temp))
	} else {
		log.Println("complete camunda task: ", completeRequest, string(pl))
	}
	return
}

type TaskInfo struct {
	WorkerId            string `json:"worker_id"`
	TaskId              string `json:"task_id"`
	ProcessInstanceId   string `json:"process_instance_id"`
	ProcessDefinitionId string `json:"process_definition_id"`
	CompletionStrategy  string `json:"completion_strategy"`
	Time                string `json:"time"`
	TenantId            string `json:"tenant_id"`
}

type CamundaCompleteRequest struct {
	WorkerId string `json:"workerId,omitempty"`
}

const userToken = "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICIzaUtabW9aUHpsMmRtQnBJdS1vSkY4ZVVUZHh4OUFIckVOcG5CcHM5SjYwIn0.eyJqdGkiOiIzMmE1OTljZC0zNDgxLTQzYWUtYWY0NC04YTVmNjU4NzYxZTUiLCJleHAiOjE1NjI5MjAwMDUsIm5iZiI6MCwiaWF0IjoxNTYyOTE2NDA1LCJpc3MiOiJodHRwczovL2F1dGguc2VwbC5pbmZhaS5vcmcvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJlYmJhZDkyNy00YzM5LTRkMTItODY5MC04OWIwNjdkZDRjZTciLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiNTVlMzA4N2UtZjljNi00MmQ2LWE0MmEtMGZiMjcxNWE4OTkyIiwiYXV0aF90aW1lIjoxNTYyOTE2NDA0LCJzZXNzaW9uX3N0YXRlIjoiYmU5MDQ2MmYtOGE3Yy00NWU4LTg1MjAtMGRlYzViZWI1ZWZlIiwiYWNyIjoiMSIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJ1bWFfYXV0aG9yaXphdGlvbiIsInVzZXIiXX0sInJlc291cmNlX2FjY2VzcyI6eyJhY2NvdW50Ijp7InJvbGVzIjpbIm1hbmFnZS1hY2NvdW50IiwibWFuYWdlLWFjY291bnQtbGlua3MiLCJ2aWV3LXByb2ZpbGUiXX19LCJyb2xlcyI6WyJ1bWFfYXV0aG9yaXphdGlvbiIsInVzZXIiLCJvZmZsaW5lX2FjY2VzcyJdLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJpbmdvIn0.pggKYb3V0VxFINWBqpFE_t14MKhSM7bhw8YqrYBRvOzh8ft7zu_-bOvLOYbJBwo0GU1D68U2d_eerkYEIt-mc0dNtdFasy5DG_GtvnWA4nsbf0BVsYKSZcRiDK4d4qbHu9NMjBdEwSkP9KDGEtou0yHtOnVzB1eHHNm_uSUO-O_kz2LWsXOPK2sbL1LTiCKS0XToJPdlaNczDMZB0nXR3sHbyi3Lwk-Va2ATS6Kke5M1KmFMowK-Y0jK2urt8GnCBIXvZMT6gUW9-dvlv4w_lAuVXQ9hFg_r0sBnoWzZOUR_xlrz2T-syjrZzmXlAkJrcD8KWPH-lCs0jD9pdiROhQ"
const userId = "ebbad927-4c39-4d12-8690-89b067dd4ce7"
const adminToken = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJqdGkiOiIzMmE1OTljZC0zNDgxLTQzYWUtYWY0NC04YTVmNjU4NzYxZTUiLCJleHAiOjE1NjI5MjAwMDUsIm5iZiI6MCwiaWF0IjoxNTYyOTE2NDA1LCJpc3MiOiJodHRwczovL2F1dGguc2VwbC5pbmZhaS5vcmcvYXV0aC9yZWFsbXMvbWFzdGVyIiwiYXVkIjoiZnJvbnRlbmQiLCJzdWIiOiJlYmJhZDkyNy00YzM5LTRkMTItODY5MC04OWIwNjdkZDRjZTciLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJmcm9udGVuZCIsIm5vbmNlIjoiNTVlMzA4N2UtZjljNi00MmQ2LWE0MmEtMGZiMjcxNWE4OTkyIiwiYXV0aF90aW1lIjoxNTYyOTE2NDA0LCJzZXNzaW9uX3N0YXRlIjoiYmU5MDQ2MmYtOGE3Yy00NWU4LTg1MjAtMGRlYzViZWI1ZWZlIiwiYWNyIjoiMSIsImFsbG93ZWQtb3JpZ2lucyI6WyIqIl0sInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJhZG1pbiIsInVtYV9hdXRob3JpemF0aW9uIiwidXNlciJdfSwicmVzb3VyY2VfYWNjZXNzIjp7ImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInJvbGVzIjpbInVtYV9hdXRob3JpemF0aW9uIiwidXNlciIsIm9mZmxpbmVfYWNjZXNzIl0sInByZWZlcnJlZF91c2VybmFtZSI6ImluZ28ifQ.k_sCFMPGvx6pXF9MU7llKSbrh3PL6OSY4PnBMvhgVKo"
