package controllers

import (
	config "TenderApi/conf"
	"database/sql"
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

type planPriceConflict struct {
	Code            string
	ExistingPrice   float64
	IncomingPrice   float64
	ItemName        string
	Mark            string
	Brand           string
	Unit            string
	Department      string
	PackageNo       string
	ExistingPackage string
}

func servePlanPriceConflict(c *MoveBasketItems, conflict planPriceConflict) {
	c.Ctx.Output.SetStatus(http.StatusConflict)
	c.Data["json"] = map[string]interface{}{
		"error":               "PLAN_PRICE_CONFLICT",
		"message":             fmt.Sprintf("%s кодтой захиалгын оруулах гэж буй нэгж үнэ %.2f ₮ нь төлөвлөгөөнд байгаа %.2f ₮ үнээс өөр байна", conflict.Code, conflict.IncomingPrice, conflict.ExistingPrice),
		"code":                conflict.Code,
		"existing_price":      conflict.ExistingPrice,
		"incoming_price":      conflict.IncomingPrice,
		"item_name":           conflict.ItemName,
		"mark":                conflict.Mark,
		"brand":               conflict.Brand,
		"unit":                conflict.Unit,
		"department":          conflict.Department,
		"package_no":          conflict.PackageNo,
		"existing_package_no": conflict.ExistingPackage,
	}
	c.ServeJSON()
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
	var targetPlanRootNumber int64
	err = tx.QueryRow(`
		SELECT COUNT(1), ISNULL(MAX(PlanRootNumber), 0)
		FROM `+schema+`.[Basket]
		WHERE BasketId = @p1 AND UserId = @p2 AND ISNULL(IsTemp, 0) = 0
	`, input.TargetBasketID, input.UserID).Scan(&targetCount, &targetPlanRootNumber)
	if err != nil || targetCount != 1 {
		c.CustomAbort(http.StatusBadRequest, "Target basket is invalid")
		return
	}

	var lockResult int
	if err := tx.QueryRow(`
		DECLARE @result int;
		EXEC @result = sp_getapplock
			@Resource = @p1,
			@LockMode = 'Exclusive',
			@LockOwner = 'Transaction',
			@LockTimeout = 10000;
		SELECT @result;
	`, fmt.Sprintf("TenderPlanPrice:%d:%d", input.UserID, targetPlanRootNumber)).Scan(&lockResult); err != nil || lockResult < 0 {
		c.CustomAbort(http.StatusConflict, "Could not lock plan price validation")
		return
	}

	placeholders := make([]string, len(input.BasketItemIDs))
	args := make([]interface{}, 0, len(input.BasketItemIDs)+2)
	args = append(args, input.TargetBasketID, input.UserID)
	for index, id := range input.BasketItemIDs {
		placeholders[index] = fmt.Sprintf("@p%d", index+3)
		args = append(args, id)
	}

	selectedPlaceholders := strings.Join(placeholders, ",")
	var conflict planPriceConflict
	internalConflictQuery := `
		SELECT TOP (1)
			LTRIM(RTRIM(ISNULL(item.Code, ''))) AS Code,
			MIN(ISNULL(item.Price, 0)) AS ExistingPrice,
			MAX(ISNULL(item.Price, 0)) AS IncomingPrice,
			MAX(ISNULL(item.Cr4Name, '')) AS ItemName,
			MAX(ISNULL(item.CrMarkName, '')) AS Mark,
			MAX(ISNULL(item.CrBrandName, '')) AS Brand,
			MAX(ISNULL(item.MName, '')) AS Unit,
			MAX(ISNULL(item.DName, '')) AS Department,
			MAX(ISNULL(item.PkgNo, '')) AS PackageNo,
			MAX(ISNULL(item.PkgNo, '')) AS ExistingPackageNo
		FROM ` + schema + `.[BasketItems] AS item
		INNER JOIN ` + schema + `.[Basket] AS sourceBasket ON sourceBasket.BasketId = item.BasketId
		WHERE sourceBasket.UserId = @p2
			AND ISNULL(item.IsReturned, 0) = 0
			AND item.BasketItemId IN (` + selectedPlaceholders + `)
			AND LTRIM(RTRIM(ISNULL(item.Code, ''))) <> ''
		GROUP BY LTRIM(RTRIM(ISNULL(item.Code, '')))
		HAVING MIN(ISNULL(item.Price, 0)) <> MAX(ISNULL(item.Price, 0))
	`
	if err := tx.QueryRow(internalConflictQuery, args...).Scan(
		&conflict.Code,
		&conflict.ExistingPrice,
		&conflict.IncomingPrice,
		&conflict.ItemName,
		&conflict.Mark,
		&conflict.Brand,
		&conflict.Unit,
		&conflict.Department,
		&conflict.PackageNo,
		&conflict.ExistingPackage,
	); err == nil {
		servePlanPriceConflict(c, conflict)
		return
	} else if err != sql.ErrNoRows {
		beego.Error("MoveBasketItems selected price validation failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to validate selected item prices")
		return
	}

	existingConflictQuery := `
		SELECT TOP (1)
			LTRIM(RTRIM(ISNULL(sourceItem.Code, ''))) AS Code,
			ISNULL(existingItem.Price, 0) AS ExistingPrice,
			ISNULL(sourceItem.Price, 0) AS IncomingPrice,
			ISNULL(sourceItem.Cr4Name, '') AS ItemName,
			ISNULL(sourceItem.CrMarkName, '') AS Mark,
			ISNULL(sourceItem.CrBrandName, '') AS Brand,
			ISNULL(sourceItem.MName, '') AS Unit,
			ISNULL(sourceItem.DName, '') AS Department,
			ISNULL(sourceItem.PkgNo, '') AS PackageNo,
			ISNULL(existingItem.PkgNo, '') AS ExistingPackageNo
		FROM ` + schema + `.[BasketItems] AS sourceItem
		INNER JOIN ` + schema + `.[Basket] AS sourceBasket ON sourceBasket.BasketId = sourceItem.BasketId
		INNER JOIN ` + schema + `.[BasketItems] AS existingItem
			ON LTRIM(RTRIM(ISNULL(existingItem.Code, ''))) = LTRIM(RTRIM(ISNULL(sourceItem.Code, '')))
		INNER JOIN ` + schema + `.[Basket] AS existingBasket ON existingBasket.BasketId = existingItem.BasketId
		WHERE sourceBasket.UserId = @p2
			AND sourceItem.BasketItemId IN (` + selectedPlaceholders + `)
			AND existingItem.BasketItemId NOT IN (` + selectedPlaceholders + `)
			AND existingBasket.UserId = @p2
			AND existingBasket.PlanRootNumber = @p1
			AND ISNULL(existingItem.IsReturned, 0) = 0
			AND ISNULL(sourceItem.IsReturned, 0) = 0
			AND LTRIM(RTRIM(ISNULL(sourceItem.Code, ''))) <> ''
			AND ISNULL(existingItem.Price, 0) <> ISNULL(sourceItem.Price, 0)
	`
	conflictArgs := make([]interface{}, 0, len(input.BasketItemIDs)+2)
	conflictArgs = append(conflictArgs, targetPlanRootNumber, input.UserID)
	for _, id := range input.BasketItemIDs {
		conflictArgs = append(conflictArgs, id)
	}
	conflict = planPriceConflict{}
	if err := tx.QueryRow(existingConflictQuery, conflictArgs...).Scan(
		&conflict.Code,
		&conflict.ExistingPrice,
		&conflict.IncomingPrice,
		&conflict.ItemName,
		&conflict.Mark,
		&conflict.Brand,
		&conflict.Unit,
		&conflict.Department,
		&conflict.PackageNo,
		&conflict.ExistingPackage,
	); err == nil {
		servePlanPriceConflict(c, conflict)
		return
	} else if err != sql.ErrNoRows {
		beego.Error("MoveBasketItems existing price validation failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to validate plan item prices")
		return
	}

	result, err := tx.Exec(`
		UPDATE item
		SET item.BasketId = @p1
		FROM `+schema+`.[BasketItems] AS item
		INNER JOIN `+schema+`.[Basket] AS sourceBasket
			ON sourceBasket.BasketId = item.BasketId
		WHERE sourceBasket.UserId = @p2
			AND (
				ISNULL(sourceBasket.IsTemp, 0) = 1
				OR UPPER(LTRIM(RTRIM(ISNULL(sourceBasket.BasketType, '')))) = 'PLAN_INBOX'
			)
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
		c.CustomAbort(http.StatusConflict, "Some basket items were not in the user's basket or plan inbox")
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
