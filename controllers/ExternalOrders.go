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
	"time"

	"github.com/astaxie/beego"
)

type ExternalOrders struct {
	beego.Controller
}

var externalOrdersClient = &http.Client{Timeout: 30 * time.Second}

const externalOrdersBackURL = "http://172.30.30.10:8008/v1/orders/back"
const externalOrdersListURL = "http://172.30.30.10:8008/v1/orders/list"

func externalOrdersBaseURL() string {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("ORDERS_API_URL")), "/"); value != "" {
		return value
	}
	return "http://192.168.4.107:8008/v1/orders"
}

func externalOrdersTokenURL() string {
	if value := strings.TrimSpace(os.Getenv("ORDERS_TOKEN_URL")); value != "" {
		return value
	}
	return "http://192.168.4.107:8008/v1/token/gtoken"
}

func fetchExternalOrdersToken() (string, error) {
	username := strings.TrimSpace(os.Getenv("ORDERS_API_USERNAME"))
	password := os.Getenv("ORDERS_API_PASSWORD")
	if username == "" || password == "" {
		return "", fmt.Errorf("orders API credentials are not configured")
	}

	body, _ := json.Marshal(map[string]string{"Username": username, "Password": password})
	request, err := http.NewRequest(http.MethodPost, externalOrdersTokenURL(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := externalOrdersClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("orders token API returned %d", response.StatusCode)
	}

	var result struct {
		Token string `json:"tokendata"`
	}
	if err := json.Unmarshal(responseBody, &result); err != nil || result.Token == "" {
		return "", fmt.Errorf("orders token API returned an invalid response")
	}
	return result.Token, nil
}

func sendExternalOrdersRequest(path string, payload map[string]interface{}) (int, []byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, nil, err
	}
	requestURL := externalOrdersBaseURL() + path
	if path == "/list" {
		requestURL = externalOrdersListURL
	} else if path == "/back" {
		requestURL = externalOrdersBackURL
	}
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := externalOrdersClient.Do(request)
	if err != nil {
		return 0, nil, err
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, err
	}
	return response.StatusCode, responseBody, nil
}

func (c *ExternalOrders) proxy(path string, withToken bool) {
	var payload map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &payload); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	delete(payload, "Token")
	if path == "/back" {
		claims, err := ClaimsForController(&c.Controller)
		if err != nil {
			beego.Error("Orders back authentication failed:", err)
			c.CustomAbort(http.StatusUnauthorized, "Invalid user session")
			return
		}
		db := connectDB(getConfig(config.Env))
		defer db.Close()
		schema := "[Tender].[dbo]"
		if config.Env == "prod" {
			schema = "[Tender].[logtender]"
		}
		var userCode string
		if err := db.QueryRow(`SELECT CONVERT(varchar(50), Code) FROM `+schema+`.[Users] WHERE Id = @p1`, claims.UserID).Scan(&userCode); err != nil {
			beego.Error("Orders back user code lookup failed: userId=", claims.UserID, "error=", err)
			c.CustomAbort(http.StatusForbidden, "Authenticated user has no order code")
			return
		}
		payload["Man"] = userCode
		beego.Info(
			"Orders back prepared: upstream=", externalOrdersBackURL,
			"pkgNo=", strings.TrimSpace(fmt.Sprint(payload["PkgNo"])),
			"pkgDate=", strings.TrimSpace(fmt.Sprint(payload["PkgDate"])),
			"man=", userCode,
			"cancel=", payload["Cancel"],
			"comment=", strings.TrimSpace(fmt.Sprint(payload["Comment"])),
		)
	}

	if withToken {
		token, err := fetchExternalOrdersToken()
		if err != nil {
			beego.Error("External orders token failed:", err)
			c.CustomAbort(http.StatusServiceUnavailable, "External orders service is not configured")
			return
		}
		payload["Token"] = token
	}

	statusCode, responseBody, err := sendExternalOrdersRequest(path, payload)
	if err != nil {
		beego.Error("External orders request failed: path=", path, "error=", err)
		c.CustomAbort(http.StatusBadGateway, "External orders service is unavailable")
		return
	}
	if path == "/back" {
		beego.Info("Orders back upstream response: status=", statusCode, "body=", string(responseBody))
	}
	c.Ctx.Output.SetStatus(statusCode)
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	c.Ctx.Output.Body(responseBody)
}

func (c *ExternalOrders) List() {
	var payload map[string]interface{}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &payload); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON payload")
		return
	}
	if strings.TrimSpace(fmt.Sprint(payload["Mm"])) != "0" {
		c.proxy("/list", true)
		return
	}

	payload["Mm"] = "12"
	payload["Man"] = ""
	delete(payload, "Token")

	token, err := fetchExternalOrdersToken()
	if err != nil {
		beego.Error("External orders token failed:", err)
		c.CustomAbort(http.StatusServiceUnavailable, "External orders service is not configured")
		return
	}
	payload["Token"] = token
	statusCode, responseBody, err := sendExternalOrdersRequest("/list", payload)
	if err != nil {
		beego.Error("External yearly orders request failed:", err)
		c.CustomAbort(http.StatusBadGateway, "External orders service is unavailable")
		return
	}
	c.Ctx.Output.SetStatus(statusCode)
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	c.Ctx.Output.Body(responseBody)
}

func (c *ExternalOrders) Status()  { c.proxy("/status", false) }
func (c *ExternalOrders) Back()    { c.proxy("/back", true) }
func (c *ExternalOrders) Client()  { c.proxy("/client", true) }
func (c *ExternalOrders) Materc2() { c.proxy("/materc2", true) }
