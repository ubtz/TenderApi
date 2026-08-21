package controllers

import (
	config "TenderApi/conf"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
)

type ExternalCpos struct {
	beego.Controller
}

const cposCacheDuration = 30 * time.Minute

type cposDataCache struct {
	sync.RWMutex
	body      []byte
	expiresAt time.Time
}

var cposCodeCache cposDataCache

func externalCposCodeURL() string {
	environmentKey := "CPOS_CODE_URL_" + strings.ToUpper(strings.TrimSpace(config.Env))
	if value := strings.TrimSpace(os.Getenv(environmentKey)); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("CPOS_CODE_URL")); value != "" {
		return value
	}
	return "http://192.168.4.107:8008/v1/cpos/code"
}

func getCachedCposData(cache *cposDataCache) []byte {
	cache.RLock()
	defer cache.RUnlock()
	if len(cache.body) == 0 || time.Now().After(cache.expiresAt) {
		return nil
	}
	return append([]byte(nil), cache.body...)
}

func setCachedCposData(cache *cposDataCache, body []byte) {
	cache.Lock()
	defer cache.Unlock()
	cache.body = append([]byte(nil), body...)
	cache.expiresAt = time.Now().Add(cposCacheDuration)
}

func fetchExternalCposData(url string) ([]byte, error) {
	token, err := fetchExternalOrdersToken()
	if err != nil {
		return nil, err
	}
	payload, _ := json.Marshal(map[string]string{"Token": token})
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := externalOrdersClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("CPOS returned HTTP %d", response.StatusCode)
	}
	var result struct {
		Status bool `json:"status"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil || !result.Status {
		return nil, fmt.Errorf("CPOS returned invalid data")
	}
	return responseBody, nil
}

func (c *ExternalCpos) serve(url string, cache *cposDataCache) {
	if cachedBody := getCachedCposData(cache); cachedBody != nil {
		c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
		c.Ctx.Output.Body(cachedBody)
		return
	}

	responseBody, err := fetchExternalCposData(url)
	if err != nil {
		time.Sleep(250 * time.Millisecond)
		responseBody, err = fetchExternalCposData(url)
	}
	if err != nil {
		beego.Error("External CPOS request failed:", err)
		c.CustomAbort(http.StatusBadGateway, "CPOS service is unavailable")
		return
	}
	setCachedCposData(cache, responseBody)
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	c.Ctx.Output.Body(responseBody)
}

func (c *ExternalCpos) Post() {
	c.serve(externalCposCodeURL(), &cposCodeCache)
}
