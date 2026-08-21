package controllers

import (
	config "TenderApi/conf"
	"net/http"
	"strconv"

	"github.com/astaxie/beego"
)

type DeleteBasketItem struct {
	beego.Controller
}

func (c *DeleteBasketItem) DeleteBasketItem() {
	// Get the "id" parameter from query
	idStr := c.GetString("id")
	beego.Info("🟢 Received id param:", idStr) // Log for debugging

	id, err := strconv.Atoi(idStr)
	if err != nil {
		beego.Error("❌ Failed to convert id:", idStr, "error:", err)
		c.CustomAbort(400, "Invalid BasketItemId")
		return
	}

	beego.Info("✅ Parsed BasketItemId:", id)
	// Connect to database
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID == 0 {
		c.CustomAbort(http.StatusUnauthorized, "Invalid user session")
		return
	}

	query := `
		DELETE item
		FROM [Tender].[dbo].[BasketItems] AS item
		INNER JOIN [Tender].[dbo].[Basket] AS basket ON basket.BasketId = item.BasketId
		WHERE item.BasketItemId = @p1 AND basket.UserId = @p2
	`

	// If prod DB uses different schema
	if config.Env == "prod" {
		query = `
			DELETE item
			FROM [Tender].[logtender].[BasketItems] AS item
			INNER JOIN [Tender].[logtender].[Basket] AS basket ON basket.BasketId = item.BasketId
			WHERE item.BasketItemId = @p1 AND basket.UserId = @p2
		`
	}

	// Execute delete
	result, err := db.Exec(query, id, claims.UserID)
	if err != nil {
		c.CustomAbort(500, "Failed to delete BasketItem: "+err.Error())
		return
	}

	// Check affected rows
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.CustomAbort(404, "No BasketItem found with the given ID")
		return
	}

	// Success response
	c.Data["json"] = map[string]interface{}{
		"success": true,
		"message": "BasketItem deleted successfully",
	}
	c.ServeJSON()
}
