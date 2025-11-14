package controllers

import (
	config "TenderApi/conf"
	"fmt"

	"github.com/astaxie/beego"
)

type GetBasketWithTotalPrice struct {
	beego.Controller
}

type BasketWithTotalPrice struct {
	BasketId       int      `json:"basket_id"`
	UserId         int      `json:"user_id"`
	AddedAt        string   `json:"added_at"`
	BasketName     string   `json:"basket_name"`
	BasketNumber   int      `json:"basket_number"`
	BasketType     string   `json:"basket_type"`
	PublishDate    string   `json:"publish_date"`
	PlanName       string   `json:"plan_name"`
	PlanRootNumber int      `json:"plan_root_number"`
	SetDate        string   `json:"set_date"`
	TotalPrice     float64  `json:"total_price"`   // <-- Add this
	UniqueDnames   []string `json:"unique_dnames"` // <-- Add this
}

// GET /get/baskets
func (c *GetBasketWithTotalPrice) GetBasketWithTotalPrice() {
	fmt.Println("📥 GetBasket endpoint hit")

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Step 1: Fetch all baskets
	basketQuery := `
		SELECT
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
		basketQuery = `
		SELECT
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
	rows, err := db.Query(basketQuery)
	if err != nil {
		fmt.Println("❌ DB query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to fetch baskets"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var baskets []BasketWithTotalPrice
	for rows.Next() {
		var b BasketWithTotalPrice
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
			&b.SetDate,
		)
		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		// Step 2: Fetch total price for this basket
		err = db.QueryRow(`
			SELECT ISNULL(SUM(CAST(pricesum AS FLOAT)), 0)
			FROM [Tender].[dbo].[BasketItems]
			WHERE BasketId = @p1`, b.BasketId).Scan(&b.TotalPrice)
		if err != nil {
			fmt.Println("⚠️ Failed to get total price for BasketId", b.BasketId, ":", err)
			b.TotalPrice = 0
		}
		if config.Env == "prod" {
			err = db.QueryRow(`
			SELECT ISNULL(SUM(CAST(pricesum AS FLOAT)), 0)
			FROM [Tender].[logtender].[BasketItems]
			WHERE BasketId = @p1`, b.BasketId).Scan(&b.TotalPrice)
			if err != nil {
				fmt.Println("⚠️ Failed to get total price for BasketId", b.BasketId, ":", err)
				b.TotalPrice = 0
			}
		}
		// Step 3: Fetch unique dnames for this basket
		dnameRows, err := db.Query(`
			SELECT DISTINCT dname
			FROM [Tender].[dbo].[BasketItems]
			WHERE BasketId = @p1`, b.BasketId)
		if config.Env == "prod" {
			dnameRows, err = db.Query(`
			SELECT DISTINCT dname
			FROM [Tender].[logtender].[BasketItems]
			WHERE BasketId = @p1`, b.BasketId)
		}
		if err != nil {
			fmt.Println("⚠️ Failed to get dnames for BasketId", b.BasketId, ":", err)
		} else {
			var dnames []string
			for dnameRows.Next() {
				var dname string
				if err := dnameRows.Scan(&dname); err == nil {
					dnames = append(dnames, dname)
				}
			}
			dnameRows.Close()
			b.UniqueDnames = dnames
		}

		baskets = append(baskets, b)
	}

	c.Data["json"] = baskets
	c.ServeJSON()
}
