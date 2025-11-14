package controllers

import (
	"encoding/json"
	"log"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
	_ "github.com/denisenkom/go-mssqldb"
)

type BasketItemController struct {
	beego.Controller
}

type UpdateBasketItemRequest struct {
	BasketItemId int64   `json:"basket_item_id"`
	NewPrice     float64 `json:"new_price"`
	NewQty       float64 `json:"new_qty"`
	NewPriceSum  float64 `json:"new_pricesum"`
	ChangeReason string  `json:"change_reason"`
}

func (c *BasketItemController) UpdateBasketItemById() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	var req UpdateBasketItemRequest
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		log.Println("JSON decode error:", err)
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "Invalid request body"}
		c.ServeJSON()
		return
	}

	query := `
		UPDATE [Tender].[dbo].[BasketItems]
		SET NewPrice = @p1, NewQty = @p2, NewPriceSum = @p3, ChangeReason = @p4
		WHERE BasketItemId = @p5
	`
	if config.Env == "prod" {
		query = `
		UPDATE [Tender].[logtender].[BasketItems]
		SET NewPrice = @p1, NewQty = @p2, NewPriceSum = @p3, ChangeReason = @p4
		WHERE BasketItemId = @p5
	`
	}
	_, err := db.Exec(query, req.NewPrice, req.NewQty, req.NewPriceSum, req.ChangeReason, req.BasketItemId)
	if err != nil {
		log.Println("DB update error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to update basket item"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]string{"message": "Basket item updated successfully"}
	c.ServeJSON()
}
