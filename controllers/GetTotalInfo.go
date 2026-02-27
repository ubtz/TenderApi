package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/astaxie/beego"
)

const (
	baseURL  = "http://192.168.4.107:8008"
	username = "coss.api.lib.nrp"
	password = "coss.api.lib.nrp."
)

var mans = []int{
	1520, 1522, 1530, 1526, 1527, 1529, 1525, 1519,
	1523, 1528, 1524, 1624, 1655, 1657, 1654,
}

type GetTotalInfo struct {
	beego.Controller
}

func getToken() (string, error) {
	payload := map[string]string{
		"Username": username,
		"Password": password,
	}

	body, _ := json.Marshal(payload)

	resp, err := http.Post(baseURL+"/v1/token/gtoken", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rawBody, _ := ioutil.ReadAll(resp.Body)
	beego.Info("TOKEN RAW RESPONSE:", string(rawBody))

	var raw map[string]interface{}
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return "", err
	}

	token, ok := raw["tokendata"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("token not found in response")
	}

	return token, nil
}

func getOrdersRaw(yy, mm, man, token string) (map[string]interface{}, error) {
	// 🔥 EXACT SAME SHAPE AS AXIOS
	payload := map[string]interface{}{
		"Yy":    yy,  // string
		"Mm":    mm,  // string
		"Man":   man, // string (your frontend sends String(user?.code))
		"Token": token,
	}

	body, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequest("POST", baseURL+"/v1/orders/list", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, _ := ioutil.ReadAll(resp.Body)
	beego.Info("ORDERS RAW RESPONSE:", string(rawBody))

	var raw map[string]interface{}
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return nil, err
	}

	return raw, nil
}

func (c *GetTotalInfo) GetTotalInfo() {
	yy := c.GetString("yy")
	if yy == "" {
		yy = "2025"
	}

	token, err := getToken()
	if err != nil {
		c.Data["json"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}

	result := make(map[string]map[string]interface{})

	for _, man := range mans {
		manStr := fmt.Sprintf("%d", man)
		result[manStr] = make(map[string]interface{})

		for mm := 1; mm <= 12; mm++ {
			mmStr := fmt.Sprintf("%d", mm) // frontend uses String(mm)

			raw, err := getOrdersRaw(yy, mmStr, manStr, token)
			if err != nil {
				result[manStr][mmStr] = map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				}
				continue
			}
			result[manStr][mmStr] = raw
		}
	}

	c.Data["json"] = map[string]interface{}{
		"success": true,
		"year":    yy,
		"data":    result,
	}
	c.ServeJSON()
}
