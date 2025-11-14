package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

// Controller
type DeleteBasket struct {
	beego.Controller
}

// Input JSON structure
type DeleteBasketInput struct {
	UserId         int    `json:"user_id"`
	PlanRootNumber int    `json:"plan_root_number,omitempty"`
	BasketType     string `json:"basket_type,omitempty"`
	BasketId       int    `json:"basket_id,omitempty"`
}

// POST /delete/deleteBasket
func (c *DeleteBasket) DeleteBasket() {
	start := time.Now()
	beego.Info("🗑️ DeleteBasket endpoint hit at", start.Format(time.RFC3339))

	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		beego.Error("❌ Empty request body")
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	// Parse JSON
	var input DeleteBasketInput
	if err := json.Unmarshal(body, &input); err != nil {
		beego.Error("❌ Invalid JSON:", err)
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON"}
		c.ServeJSON()
		return
	}

	// Validate
	if input.UserId == 0 {
		beego.Error("❌ Missing user_id")
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Missing user_id"}
		c.ServeJSON()
		return
	}

	// Log parsed data
	beego.Info(fmt.Sprintf("🧾 Delete request from user=%d, plan=%d, type=%q, basket=%d",
		input.UserId, input.PlanRootNumber, input.BasketType, input.BasketId))

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	var deleteQuery string
	var args []interface{}
	var deleteScope string

	switch {
	case input.BasketId != 0:
		// Single basket
		deleteScope = "basket"
		deleteQuery = `
			DELETE FROM [Tender].[dbo].[Basket]
			WHERE BasketId = @p1 AND UserId = @p2
		`
		args = []interface{}{input.BasketId, input.UserId}

	case input.BasketType != "" && input.PlanRootNumber != 0:
		// All baskets in one category
		deleteScope = "category"
		deleteQuery = `
			DELETE FROM [Tender].[dbo].[Basket]
			WHERE BasketType = @p1 AND PlanRootNumber = @p2 AND UserId = @p3
		`
		args = []interface{}{input.BasketType, input.PlanRootNumber, input.UserId}

	case input.PlanRootNumber != 0:
		// Entire plan
		deleteScope = "plan"
		deleteQuery = `
			DELETE FROM [Tender].[dbo].[Basket]
			WHERE PlanRootNumber = @p1 AND UserId = @p2
		`
		args = []interface{}{input.PlanRootNumber, input.UserId}

	default:
		beego.Warn("⚠️ Invalid delete parameters received:", string(body))
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid delete parameters"}
		c.ServeJSON()
		return
	}

	if config.Env == "prod" {
		deleteQuery = strings.Replace(deleteQuery, "[Tender].[dbo]", "[Tender].[logtender]", 1)
	}

	result, err := db.Exec(deleteQuery, args...)
	if err != nil {
		beego.Error("❌ SQL delete failed:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to delete records"}
		c.ServeJSON()
		return
	}

	rows, _ := result.RowsAffected()
	elapsed := time.Since(start)

	// Log success
	beego.Info(fmt.Sprintf("✅ %s deleted successfully | user=%d | affected=%d | duration=%s",
		strings.Title(deleteScope), input.UserId, rows, elapsed))

	// JSON response
	c.Data["json"] = map[string]interface{}{
		"message":       "Deleted successfully",
		"deleted_scope": deleteScope,
		"deleted_count": rows,
	}
	c.ServeJSON()
}
