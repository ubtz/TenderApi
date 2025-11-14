package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type UpdateBaskedValidation struct {
	beego.Controller
}

// POST /updateBasketValid
// ✅ POST /updateBasketValid
func (c *UpdateBaskedValidation) UpdateBaskedValidation() {
	fmt.Println("📥 UpdateBasketValid endpoint hit")

	body := c.Ctx.Input.RequestBody
	fmt.Println("📦 Raw body:", string(body)) // 👈 log raw request body

	// Read body
	body = c.Ctx.Input.RequestBody
	if len(body) == 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	// Parse JSON { "planRootNumber": 123 }
	var input struct {
		PlanRootNumber int `json:"planRootNumber"`
	}
	if err := json.Unmarshal(body, &input); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON"}
		c.ServeJSON()
		return
	}

	// Validate
	if input.PlanRootNumber == 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "planRootNumber is required"}
		c.ServeJSON()
		return
	}

	// DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Update query
	updateQuery := `
		UPDATE [Tender].[dbo].[Basket]
		SET isValid = CAST(1 AS BIT)
		WHERE PlanRootNumber = @p1
	`
	if config.Env == "prod" {
		updateQuery = `
		UPDATE [Tender].[logtender].[Basket]
		SET isValid = CAST(1 AS BIT)
		WHERE PlanRootNumber = @p1
		`
	}

	// Execute update
	result, err := db.Exec(updateQuery, input.PlanRootNumber)
	if err != nil {
		fmt.Println("❌ Failed to update baskets:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to update baskets"}
		c.ServeJSON()
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Updated %d baskets (PlanRootNumber: %d)\n", rowsAffected, input.PlanRootNumber)

	c.Data["json"] = map[string]interface{}{
		"message":        "Baskets updated successfully",
		"rowsAffected":   rowsAffected,
		"planRootNumber": input.PlanRootNumber,
	}
	c.ServeJSON()
}
