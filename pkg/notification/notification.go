/*
 * Copyright 2022 InfAI (CC SES)
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

package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/SENERGY-Platform/gin-middleware/otelx"
)

func Send(ctx context.Context, notificationUrl string, message Message, logger *slog.Logger) error {
	if notificationUrl == "" {
		return nil
	}
	if message.Topic == "" {
		message.Topic = "smart_service"
	}
	logger.DebugContext(ctx, "send notification", "notificationUrl", notificationUrl, "message", message)
	b := new(bytes.Buffer)
	err := json.NewEncoder(b).Encode(message)
	if err != nil {
		return err
	}
	//WithoutCancel keeps trace-context and baggage, but leaves the notification unaffected
	//by a client that aborts the request it was triggered by
	timeout, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(timeout, "POST", notificationUrl+"/notifications", b)
	if err != nil {
		logger.ErrorContext(ctx, "unable to send notification", "error", err)
		return err
	}
	err = otelx.InjectContextToRequest(ctx, req)
	if err != nil {
		logger.ErrorContext(ctx, "unable to send notification", "error", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.ErrorContext(ctx, "unable to send notification", "error", err)
		return err
	}
	if resp.StatusCode >= 300 {
		respMsg, _ := io.ReadAll(resp.Body)
		err = errors.New("unexpected response status from notifier " + resp.Status)
		logger.ErrorContext(ctx, "unexpected response status from notifier", "error", err, "respMsg", string(respMsg))
		return err
	}
	return nil
}

type Message struct {
	UserId  string `json:"userId" bson:"userId"`
	Title   string `json:"title" bson:"title"`
	Message string `json:"message" bson:"message"`
	Topic   string `json:"topic" bson:"topic"`
}
