package controllers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type UpdateGereeInput struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

type UpdateGeree struct {
	beego.Controller
}

// PUT /put/UpdateGeree/:id
func (c *UpdateGeree) Put() {
	gereeID := c.Ctx.Input.Param(":id")
	if gereeID == "" {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "GereeId is required"}
		c.ServeJSON()
		return
	}

	// 🔹 Log raw body
	body, _ := ioutil.ReadAll(c.Ctx.Request.Body)
	log.Printf("📥 Raw request body: %s", string(body))

	var input UpdateGereeInput
	if err := json.Unmarshal(body, &input); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON"}
		c.ServeJSON()
		return
	}

	if input.Field == "" {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Field name is required"}
		c.ServeJSON()
		return
	}

	// DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Handle type conversion (💡 important!)
	var param interface{} = input.Value

	// If value is empty string, store NULL instead
	if str, ok := input.Value.(string); ok {
		if str == "" {
			param = nil
		}
	}

	// If field is numeric (like "Гэрээний_дүн") and input is string, convert
	if input.Field == "Гэрээний_дүн" {
		switch v := input.Value.(type) {
		case string:
			if v == "" {
				param = nil
			} else {
				if f, err := strconv.ParseFloat(v, 64); err == nil {
					param = f
				} else {
					log.Printf("⚠️ Failed to parse float for value=%v", v)
					param = nil
				}
			}
		}
	}

	// SQL query (🛡️ be careful with dynamic column names)
	query := fmt.Sprintf(`UPDATE [Tender].[dbo].[Geree] SET [%s] = @p1 WHERE GereeId = @p2`, input.Field)
	if config.Env == "prod" {
		query = fmt.Sprintf(`UPDATE [Tender].[logtender].[Geree] SET [%s] = @p1 WHERE GereeId = @p2`, input.Field)
	}
	log.Printf("📝 Executing query: %s | value=%v | id=%s", query, param, gereeID)

	_, err := db.Exec(query, param, gereeID)
	if err != nil {
		log.Printf("❌ Failed to update geree: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to update geree"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]string{"message": "Geree updated successfully"}
	c.ServeJSON()
}
