package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/astaxie/beego"
)

// Controller struct
type UpdateBasketItemState struct {
	beego.Controller
}

// Request model
type UpdateBasketItemRequestState struct {
	BasketItemId int     `json:"basket_item_id"`
	IsArrived    *bool   `json:"isArrived"`
	Tailbar      *string `json:"tailbar"`
}

// ✅ PUT /update/basketitem/state
func (c *UpdateBasketItemState) Put() {
	fmt.Println("📥 UpdateBasketItemState endpoint hit")

	var req UpdateBasketItemRequestState
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &req)
	if err != nil {
		fmt.Println("❌ JSON parse error:", err)
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if req.BasketItemId == 0 {
		c.CustomAbort(http.StatusBadRequest, "basket_item_id is required")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	table := "[Tender].[dbo].[BasketItems]"
	if config.Env == "prod" {
		table = "[Tender].[logtender].[BasketItems]"
	}

	// ✅ Build dynamic update query
	query := fmt.Sprintf("UPDATE %s SET ", table)
	var params []interface{}
	var setClauses []string

	if req.IsArrived != nil {
		setClauses = append(setClauses, "isArrived = @p1")
		params = append(params, *req.IsArrived)
	}
	if req.Tailbar != nil {
		setClauses = append(setClauses, fmt.Sprintf("Tailbar = @p%d", len(params)+1))
		params = append(params, *req.Tailbar)
	}

	if len(setClauses) == 0 {
		c.CustomAbort(http.StatusBadRequest, "No fields to update")
		return
	}

	query += fmt.Sprintf("%s WHERE BasketItemId = @p%d",
		sqlClauseJoin(setClauses, ", "), len(params)+1)
	params = append(params, req.BasketItemId)

	fmt.Println("🧾 Final Query:", query)
	fmt.Println("📦 Params:", params)

	_, err = db.Exec(query, params...)
	if err != nil {
		fmt.Println("❌ DB update error:", err)
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}

	c.Data["json"] = map[string]string{"message": "Basket item updated successfully"}
	c.ServeJSON()
}

// helper to safely join SQL clauses
func sqlClauseJoin(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep
		}
		out += p
	}
	return out
}
