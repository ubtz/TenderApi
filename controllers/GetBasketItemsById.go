package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"fmt"
	"strings"

	"github.com/astaxie/beego"
)

type GetBasketItemsById struct {
	beego.Controller
}

// 🔹 Struct for DB Scan (can handle NULL)
type BasketItemData struct {
	BasketItemId int            `json:"basket_item_id"`
	BasketId     int            `json:"basket_id"`
	BasketName   string         `json:"basket_name"`
	AddedAt      string         `json:"added_at"`
	Acct         string         `json:"acct"`
	Barcode      string         `json:"barcode"`
	DDate        string         `json:"ddate"`
	CDate        string         `json:"cdate"`
	UDate        string         `json:"udate"`
	DEDate       string         `json:"dedate"`
	Code         string         `json:"code"`
	CR1ID        string         `json:"cr1id"`
	CR1Name      string         `json:"cr1name"`
	CR2ID        string         `json:"cr2id"`
	CR3ID        string         `json:"cr3id"`
	CR4ID        string         `json:"cr4id"`
	CR4Name      string         `json:"cr4name"`
	CRBrand      string         `json:"crbrand"`
	CRBrandName  string         `json:"crbrandname"`
	CRMark       string         `json:"crmark"`
	CRMarkName   string         `json:"crmarkname"`
	DCode        string         `json:"dcode"`
	DName        string         `json:"dname"`
	MDocNo       string         `json:"mdocno"`
	MeasId       string         `json:"measid"`
	MName        string         `json:"mname"`
	Price        float64        `json:"price"`
	PriceSum     float64        `json:"pricesum"`
	Qty          float64        `json:"qty"`
	RId          string         `json:"rid"`
	USize        string         `json:"usize"`
	ZNo          string         `json:"zno"`
	IsArrived    sql.NullBool   `json:"isArrived"`
	Tailbar      sql.NullString `json:"tailbar"`
}

// 🔹 Struct for JSON Response (clean, no sql.Null*)
type BasketItemResponse struct {
	BasketItemId int     `json:"basket_item_id"`
	BasketId     int     `json:"basket_id"`
	BasketName   string  `json:"basket_name"`
	AddedAt      string  `json:"added_at"`
	Acct         string  `json:"acct"`
	Barcode      string  `json:"barcode"`
	DDate        string  `json:"ddate"`
	CDate        string  `json:"cdate"`
	UDate        string  `json:"udate"`
	DEDate       string  `json:"dedate"`
	Code         string  `json:"code"`
	CR1ID        string  `json:"cr1id"`
	CR1Name      string  `json:"cr1name"`
	CR2ID        string  `json:"cr2id"`
	CR3ID        string  `json:"cr3id"`
	CR4ID        string  `json:"cr4id"`
	CR4Name      string  `json:"cr4name"`
	CRBrand      string  `json:"crbrand"`
	CRBrandName  string  `json:"crbrandname"`
	CRMark       string  `json:"crmark"`
	CRMarkName   string  `json:"crmarkname"`
	DCode        string  `json:"dcode"`
	DName        string  `json:"dname"`
	MDocNo       string  `json:"mdocno"`
	MeasId       string  `json:"measid"`
	MName        string  `json:"mname"`
	Price        float64 `json:"price"`
	PriceSum     float64 `json:"pricesum"`
	Qty          float64 `json:"qty"`
	RId          string  `json:"rid"`
	USize        string  `json:"usize"`
	ZNo          string  `json:"zno"`
	IsArrived    *bool   `json:"isArrived,omitempty"`
	Tailbar      *string `json:"tailbar,omitempty"`
}

// ✅ GET /get/basketitems/:basketId
func (c *GetBasketItemsById) GetBasketItemsById() {
	fmt.Println("📥 GetBasketItemsById endpoint hit")

	basketIDsParam := c.Ctx.Input.Param(":basketId")
	if basketIDsParam == "" {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "basket_id is required"}
		c.ServeJSON()
		return
	}

	// Split and trim
	basketIDs := strings.Split(basketIDsParam, ",")
	for i, id := range basketIDs {
		basketIDs[i] = strings.TrimSpace(id)
	}

	// Prepare placeholders and args
	var placeholders []string
	var args []interface{}
	for i, id := range basketIDs {
		paramName := fmt.Sprintf("@p%d", i+1)
		placeholders = append(placeholders, paramName)
		args = append(args, id)
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	basketTable := "[Tender].[dbo].[Basket]"
	basketItemsTable := "[Tender].[dbo].[BasketItems]"
	if config.Env == "prod" {
		basketTable = "[Tender].[logtender].[Basket]"
		basketItemsTable = "[Tender].[logtender].[BasketItems]"
	}

	// ✅ JOIN query with Basket table to include BasketName
	query := fmt.Sprintf(`
        SELECT 
            bi.BasketItemId,
            bi.BasketId,
            b.BasketName,
            bi.AddedAt,
            bi.acct,
            bi.barcode,
            bi.ddate,
            bi.cdate,
            bi.udate,
            bi.dedate,
            bi.code,
            bi.cr1id,
            bi.cr1name,
            bi.cr2id,
            bi.cr3id,
            bi.cr4id,
            bi.cr4name,
            bi.crbrand,
            bi.crbrandname,
            bi.crmark,
            bi.crmarkname,
            bi.dcode,
            bi.dname,
            bi.mdocno,
            bi.measid,
            bi.mname,
            bi.price,
            bi.pricesum,
            bi.qty,
            bi.rid,
            bi.usize,
            bi.zno,
            bi.isArrived,
            bi.tailbar
        FROM %s bi
        INNER JOIN %s b ON bi.BasketId = b.BasketId
        WHERE bi.BasketId IN (%s)
    `, basketItemsTable, basketTable, strings.Join(placeholders, ","))

	fmt.Println("🧾 Final Query:", query)
	fmt.Println("📦 Args:", args)

	rows, err := db.Query(query, args...)
	if err != nil {
		fmt.Println("❌ DB query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": err.Error()}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var basketItems []BasketItemResponse
	for rows.Next() {
		var bi BasketItemData
		err := rows.Scan(
			&bi.BasketItemId,
			&bi.BasketId,
			&bi.BasketName,
			&bi.AddedAt,
			&bi.Acct,
			&bi.Barcode,
			&bi.DDate,
			&bi.CDate,
			&bi.UDate,
			&bi.DEDate,
			&bi.Code,
			&bi.CR1ID,
			&bi.CR1Name,
			&bi.CR2ID,
			&bi.CR3ID,
			&bi.CR4ID,
			&bi.CR4Name,
			&bi.CRBrand,
			&bi.CRBrandName,
			&bi.CRMark,
			&bi.CRMarkName,
			&bi.DCode,
			&bi.DName,
			&bi.MDocNo,
			&bi.MeasId,
			&bi.MName,
			&bi.Price,
			&bi.PriceSum,
			&bi.Qty,
			&bi.RId,
			&bi.USize,
			&bi.ZNo,
			&bi.IsArrived,
			&bi.Tailbar,
		)
		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		// Convert sql.Null* → normal pointers
		res := BasketItemResponse{
			BasketItemId: bi.BasketItemId,
			BasketId:     bi.BasketId,
			BasketName:   bi.BasketName,
			AddedAt:      bi.AddedAt,
			Acct:         bi.Acct,
			Barcode:      bi.Barcode,
			DDate:        bi.DDate,
			CDate:        bi.CDate,
			UDate:        bi.UDate,
			DEDate:       bi.DEDate,
			Code:         bi.Code,
			CR1ID:        bi.CR1ID,
			CR1Name:      bi.CR1Name,
			CR2ID:        bi.CR2ID,
			CR3ID:        bi.CR3ID,
			CR4ID:        bi.CR4ID,
			CR4Name:      bi.CR4Name,
			CRBrand:      bi.CRBrand,
			CRBrandName:  bi.CRBrandName,
			CRMark:       bi.CRMark,
			CRMarkName:   bi.CRMarkName,
			DCode:        bi.DCode,
			DName:        bi.DName,
			MDocNo:       bi.MDocNo,
			MeasId:       bi.MeasId,
			MName:        bi.MName,
			Price:        bi.Price,
			PriceSum:     bi.PriceSum,
			Qty:          bi.Qty,
			RId:          bi.RId,
			USize:        bi.USize,
			ZNo:          bi.ZNo,
		}

		if bi.IsArrived.Valid {
			res.IsArrived = &bi.IsArrived.Bool
		}
		if bi.Tailbar.Valid {
			res.Tailbar = &bi.Tailbar.String
		}

		basketItems = append(basketItems, res)
	}

	c.Data["json"] = basketItems
	c.ServeJSON()
}
