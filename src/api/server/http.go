package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"time"
)


// createHTTPClient создает объект HTTP клиента, который вызывается в приложении
// если нужно совершить HTTP запрос
func createHTTPClient() *http.Client {
	client := &http.Client{
		Timeout: time.Duration(10) * time.Second,
	}
	return client
}

// DoAuthRequestWithObj совершает авторизированный запрос к серверам Яндекса по определенным
// параметрам: ссылке, методу и объекту, который передается в запросе
func DoAuthRequestWithObj(method string, URL string, obj interface{}) *http.Response{
	jsonObj, err := json.Marshal(obj)
	if err != nil {
		log.WithFields(log.Fields{
			"function" : "sendShipmentsInfo",
		},
		).Warn("Can't retrieve open orders!")
	}
	req, err := http.NewRequest(method, URL, bytes.NewBuffer(jsonObj))
	req.Header.Set("Authorization", fmt.Sprintf("OAuth oauth_token=%s, oauth_client_id=%s", cfg.Beru.OAuthToken, cfg.Beru.OAuthClientID))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"function": "DoAuthRequestWithObj",
			"method":   method,
			"url":      URL,
			"error":    err,
		},
		).Warn("Request failed!")
		return nil
	}
	return resp
}

// DoAuthRequest совершает авторизированный запрос к серверам Яндекса по определенным
// параметрам: ссылке, методу и телу запроса
func DoAuthRequest(method string, URL string, body io.Reader) *http.Response{
	req, err := http.NewRequest(method, URL, body)
	req.Header.Set("Authorization", fmt.Sprintf("OAuth oauth_token=%s, oauth_client_id=%s", cfg.Beru.OAuthToken, cfg.Beru.OAuthClientID))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		log.WithFields(log.Fields{
			"function": "DoAuthRequest",
			"method":   method,
			"url":      URL,
			"error":    err,
		},
		).Warn("Request failed!")
		return nil
	}
	return resp
}