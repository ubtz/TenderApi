package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/astaxie/beego"
)

type ExternalEmployee struct {
	beego.Controller
}

func (c *ExternalEmployee) Post() {
	var input struct {
		Regno string `json:"regno"`
	}
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil || strings.TrimSpace(input.Regno) == "" {
		c.CustomAbort(http.StatusBadRequest, "regno is required")
		return
	}

	baseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("EMPLOYEE_API_URL")), "/")
	username := strings.TrimSpace(os.Getenv("EMPLOYEE_API_USERNAME"))
	password := os.Getenv("EMPLOYEE_API_PASSWORD")
	if baseURL == "" || username == "" || password == "" {
		c.CustomAbort(http.StatusServiceUnavailable, "Employee service is not configured")
		return
	}

	client := &http.Client{Timeout: 20 * time.Second}
	loginBody, _ := json.Marshal(map[string]string{"username": username, "password": password})
	loginRequest, _ := http.NewRequest(http.MethodPost, baseURL+"/external/login", bytes.NewReader(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginResponse, err := client.Do(loginRequest)
	if err != nil {
		c.CustomAbort(http.StatusBadGateway, "Employee service is unavailable")
		return
	}
	defer loginResponse.Body.Close()
	var loginResult struct {
		Token string `json:"token"`
	}
	if loginResponse.StatusCode < 200 || loginResponse.StatusCode >= 300 || json.NewDecoder(loginResponse.Body).Decode(&loginResult) != nil || loginResult.Token == "" {
		c.CustomAbort(http.StatusBadGateway, "Employee service login failed")
		return
	}

	employeeBody, _ := json.Marshal(map[string]string{"regno": strings.TrimSpace(input.Regno)})
	employeeRequest, _ := http.NewRequest(http.MethodPost, baseURL+"/api/tender/employee", bytes.NewReader(employeeBody))
	employeeRequest.Header.Set("Content-Type", "application/json")
	employeeRequest.Header.Set("Authorization", "Bearer "+loginResult.Token)
	employeeResponse, err := client.Do(employeeRequest)
	if err != nil {
		c.CustomAbort(http.StatusBadGateway, "Employee service is unavailable")
		return
	}
	defer employeeResponse.Body.Close()
	responseBody, err := io.ReadAll(employeeResponse.Body)
	if err != nil {
		c.CustomAbort(http.StatusBadGateway, "Failed to read employee response")
		return
	}
	c.Ctx.Output.SetStatus(employeeResponse.StatusCode)
	c.Ctx.Output.Header("Content-Type", "application/json; charset=utf-8")
	c.Ctx.Output.Body(responseBody)
}
