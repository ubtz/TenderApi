package controllers

import (
	config "TenderApi/conf"
	"fmt"
	"net/http"
	"strconv"

	"github.com/astaxie/beego"
)

type GetBasketItems struct {
	beego.Controller
}

type BasketItem struct {
	BasketItemId int     `json:"basketItemId"`
	BasketId     int     `json:"basketId"`
	Code         string  `json:"code"`
	DName        string  `json:"dname"`
	Qty          int     `json:"qty"`
	Price        float64 `json:"price"`
	PriceSum     float64 `json:"pricesum"`
	AddedAt      string  `json:"addedAt"`
}

// GET /get/GetBasketItems?basketId=123
func (c *GetBasketItems) GetBasketItems() {
	fmt.Println("📥 GetBasketItems endpoint hit")

	basketIDStr := c.GetString("basketId")
	basketID, err := strconv.Atoi(basketIDStr)
	if err != nil || basketID <= 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid basketId"}
		c.ServeJSON()
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
        SELECT
            BasketItemId,
            BasketId,
            code,
            dname,
            qty,
            price,
            pricesum,
            AddedAt
        FROM [Tender].[dbo].[BasketItems]
        WHERE BasketId = @p1
    `
	if config.Env == "prod" {
		query = `
        SELECT
            BasketItemId,
            BasketId,
            code,
            dname,
            qty,
            price,
            pricesum,
            AddedAt
        FROM [Tender].[logtender].[BasketItems]
        WHERE BasketId = @p1
        `
	}

	rows, err := db.Query(query, basketID)
	if err != nil {
		fmt.Println("❌ DB query error:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to fetch basket items"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var items []BasketItem
	for rows.Next() {
		var item BasketItem
		err := rows.Scan(
			&item.BasketItemId,
			&item.BasketId,
			&item.Code,
			&item.DName,
			&item.Qty,
			&item.Price,
			&item.PriceSum,
			&item.AddedAt,
		)
		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}
		items = append(items, item)
	}

	c.Data["json"] = items
	c.ServeJSON()
}
