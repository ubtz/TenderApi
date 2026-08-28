package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/astaxie/beego"
)

type ReturnBasketItems struct {
	beego.Controller
}

type ReturnBasketItemsInput struct {
	BasketItemIDs []int64 `json:"basket_item_ids"`
	UserID        int64   `json:"user_id"`
}

func (c *ReturnBasketItems) Put() {
	var input ReturnBasketItemsInput
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON")
		return
	}
	if len(input.BasketItemIDs) == 0 || input.UserID == 0 {
		c.CustomAbort(http.StatusBadRequest, "basket_item_ids and user_id are required")
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

	placeholders := make([]string, len(input.BasketItemIDs))
	args := make([]interface{}, 0, len(input.BasketItemIDs)+1)
	args = append(args, input.UserID)
	for index, id := range input.BasketItemIDs {
		placeholders[index] = fmt.Sprintf("@p%d", index+2)
		args = append(args, id)
	}

	result, err := db.Exec(`
		UPDATE item
		SET item.IsReturned = 1
		FROM `+schema+`.[BasketItems] AS item
		INNER JOIN `+schema+`.[Basket] AS basket
			ON basket.BasketId = item.BasketId
		WHERE basket.UserId = @p1
			AND ISNULL(basket.IsTemp, 0) = 1
			AND ISNULL(item.IsReturned, 0) = 0
			AND item.BasketItemId IN (`+strings.Join(placeholders, ",")+`)
	`, args...)
	if err != nil {
		beego.Error("ReturnBasketItems update failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to mark basket items returned")
		return
	}

	updated, _ := result.RowsAffected()
	if updated != int64(len(input.BasketItemIDs)) {
		c.CustomAbort(http.StatusConflict, "Some basket items could not be returned")
		return
	}
	createNotificationSafe(
		db,
		claims.UserID,
		"order_returned",
		"Захиалга буцаагдлаа",
		fmt.Sprintf("%d захиалга буцаагдсан төлөвт орлоо", updated),
		"BasketItem",
		0,
		"/Багцлах",
	)

	c.Data["json"] = map[string]interface{}{
		"message":        "Basket items marked returned",
		"returned_count": updated,
	}
	c.ServeJSON()
}
