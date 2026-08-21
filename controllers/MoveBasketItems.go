package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/astaxie/beego"
)

type MoveBasketItems struct {
	beego.Controller
}

type MoveBasketItemsInput struct {
	BasketItemIDs  []int64 `json:"basket_item_ids"`
	TargetBasketID int64   `json:"target_basket_id"`
	UserID         int64   `json:"user_id"`
}

func (c *MoveBasketItems) Put() {
	var input MoveBasketItemsInput
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON")
		return
	}
	if len(input.BasketItemIDs) == 0 || input.TargetBasketID == 0 || input.UserID == 0 {
		c.CustomAbort(http.StatusBadRequest, "basket_item_ids, target_basket_id and user_id are required")
		return
	}
	claims, err := ClaimsForController(&c.Controller)
	if err != nil || int64(claims.UserID) != input.UserID {
		c.CustomAbort(http.StatusForbidden, "User does not match authenticated session")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	schema := "[Tender].[dbo]"
	if config.Env == "prod" {
		schema = "[Tender].[logtender]"
	}

	tx, err := db.Begin()
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	var targetCount int
	err = tx.QueryRow(`
		SELECT COUNT(1)
		FROM `+schema+`.[Basket]
		WHERE BasketId = @p1 AND UserId = @p2 AND ISNULL(IsTemp, 0) = 0
	`, input.TargetBasketID, input.UserID).Scan(&targetCount)
	if err != nil || targetCount != 1 {
		c.CustomAbort(http.StatusBadRequest, "Target basket is invalid")
		return
	}

	placeholders := make([]string, len(input.BasketItemIDs))
	args := make([]interface{}, 0, len(input.BasketItemIDs)+2)
	args = append(args, input.TargetBasketID, input.UserID)
	for index, id := range input.BasketItemIDs {
		placeholders[index] = fmt.Sprintf("@p%d", index+3)
		args = append(args, id)
	}

	result, err := tx.Exec(`
		UPDATE item
		SET item.BasketId = @p1
		FROM `+schema+`.[BasketItems] AS item
		INNER JOIN `+schema+`.[Basket] AS sourceBasket
			ON sourceBasket.BasketId = item.BasketId
		WHERE sourceBasket.UserId = @p2
			AND ISNULL(sourceBasket.IsTemp, 0) = 1
			AND ISNULL(item.IsReturned, 0) = 0
			AND item.BasketItemId IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		beego.Error("MoveBasketItems update failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to move basket items")
		return
	}

	moved, err := result.RowsAffected()
	if err != nil || moved != int64(len(input.BasketItemIDs)) {
		c.CustomAbort(http.StatusConflict, "Some basket items were not in the user's temporary basket")
		return
	}
	if err := tx.Commit(); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to commit transaction")
		return
	}

	c.Data["json"] = map[string]interface{}{
		"message":     "Basket items moved successfully",
		"moved_count": moved,
	}
	c.ServeJSON()
}
