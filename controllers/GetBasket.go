package controllers

import (
	config "TenderApi/conf"
	"fmt"

	"github.com/astaxie/beego"
)

type GetBasket struct {
	beego.Controller
}

type Basket struct {
	BasketId       int    `json:"basket_id"`
	UserId         int    `json:"user_id"`
	AddedAt        string `json:"added_at"`
	BasketName     string `json:"basket_name"`
	BasketNumber   int    `json:"basket_number"`
	BasketType     string `json:"basket_type"`
	PublishDate    string `json:"publish_date"`
	PlanName       string `json:"plan_name"`
	PlanRootNumber int    `json:"plan_root_number"`
	// Category       string `json:"category"` // Ангилал
	SetDate string `json:"set_date"` // SetDate field if needed
}

// GET /get/baskets
func (c *GetBasket) GetBasket() {
	fmt.Println("📥 GetBasket endpoint hit")

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
        SELECT TOP (10000)
            BasketId,
            UserId,
            AddedAt,
            BasketName,
            BasketNumber,
            BasketType,
            PublishDate,
            PlanName,
            PlanRootNumber,
			SetDate
        FROM [Tender].[dbo].[Basket]`
	if config.Env == "prod" {
		query = `
        SELECT TOP (10000)
            BasketId,
            UserId,
            AddedAt,
            BasketName,
            BasketNumber,
            BasketType,
            PublishDate,
            PlanName,
            PlanRootNumber,
			SetDate
        FROM [Tender].[logtender].[Basket]`
	}
	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("❌ DB query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to fetch baskets"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var baskets []Basket
	for rows.Next() {
		var b Basket
		err := rows.Scan(
			&b.BasketId,
			&b.UserId,
			&b.AddedAt,
			&b.BasketName,
			&b.BasketNumber,
			&b.BasketType,
			&b.PublishDate,
			&b.PlanName,
			&b.PlanRootNumber,
			&b.SetDate, // Assuming SetDate is a field in the Basket table
		)
		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}
		baskets = append(baskets, b)
	}

	c.Data["json"] = baskets
	c.ServeJSON()
}
