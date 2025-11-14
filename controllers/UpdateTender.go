package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type UpdateTenderInput struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

type UpdateTender struct {
	beego.Controller
}

// PUT /put/UpdateTender/:id
func (c *UpdateTender) Put() {
	tenderID := c.Ctx.Input.Param(":id")
	if tenderID == "" {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "TenderId is required"}
		c.ServeJSON()
		return
	}

	var input UpdateTenderInput
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil {
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

	// SQL injection хамгаалалттай query
	query := fmt.Sprintf(`UPDATE [Tender].[dbo].[Tender] SET [%s] = @p1 WHERE TenderId = @p2`, input.Field)
	if config.Env == "prod" {
		query = fmt.Sprintf(`UPDATE [Tender].[logtender].[Tender] SET [%s] = @p1 WHERE TenderId = @p2`, input.Field)
	}
	_, err := db.Exec(query, input.Value, tenderID)
	if err != nil {
		log.Printf("❌ Failed to update tender: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to update tender"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]string{"message": "Tender updated successfully"}
	c.ServeJSON()
}
