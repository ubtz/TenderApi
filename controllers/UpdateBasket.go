package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

// Controller
type UpdateBasket struct {
	beego.Controller
}

// Input struct
type UpdateBasketInput struct {
	UserId         int    `json:"user_id"`
	PlanRootNumber int    `json:"plan_root_number,omitempty"`
	BasketId       int    `json:"basket_id,omitempty"`
	BasketType     string `json:"basket_type,omitempty"`
	NewPlanName    string `json:"new_plan_name,omitempty"`
	NewTypeName    string `json:"new_type_name,omitempty"`
	NewBasketName  string `json:"new_basket_name,omitempty"`
}

// PUT /update/updateBasket
func (c *UpdateBasket) UpdateBasket() {
	fmt.Println("📝 UpdateBasket endpoint hit")

	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		c.CustomAbort(http.StatusBadRequest, "Empty request body")
	}

	var input UpdateBasketInput
	if err := json.Unmarshal(body, &input); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON")
	}

	if input.UserId == 0 {
		c.CustomAbort(http.StatusBadRequest, "Missing user_id")
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	var query string
	var args []interface{}

	switch {
	case input.NewPlanName != "" && input.PlanRootNumber != 0:
		query = `
			UPDATE [Tender].[dbo].[Basket]
			SET PlanName = @p1
			WHERE PlanRootNumber = @p2 AND UserId = @p3
		`
		args = []interface{}{input.NewPlanName, input.PlanRootNumber, input.UserId}

	case input.NewTypeName != "" && input.BasketType != "":
		query = `
			UPDATE [Tender].[dbo].[Basket]
			SET BasketType = @p1
			WHERE BasketType = @p2 AND PlanRootNumber = @p3 AND UserId = @p4
		`
		args = []interface{}{input.NewTypeName, input.BasketType, input.PlanRootNumber, input.UserId}

	case input.NewBasketName != "" && input.BasketId != 0:
		query = `
			UPDATE [Tender].[dbo].[Basket]
			SET BasketName = @p1
			WHERE BasketId = @p2 AND UserId = @p3
		`
		args = []interface{}{input.NewBasketName, input.BasketId, input.UserId}

	default:
		c.CustomAbort(http.StatusBadRequest, "Invalid update parameters")
	}

	if config.Env == "prod" {
		query = strings.Replace(query, "[Tender].[dbo]", "[Tender].[logtender]", 1)
	}

	result, err := db.Exec(query, args...)
	if err != nil {
		fmt.Println("❌ Update error:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to update record")
	}

	rows, _ := result.RowsAffected()
	fmt.Printf("✅ Updated %d record(s)\n", rows)

	c.Data["json"] = map[string]interface{}{
		"message":       "Updated successfully",
		"updated_count": rows,
	}
	c.ServeJSON()
}
