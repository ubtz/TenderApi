package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/astaxie/beego"
	_ "github.com/denisenkom/go-mssqldb"
)

// Struct with exported field names
type BasketItemPost struct {
	BasketId    int        `json:"BasketId"`
	Acct        string     `json:"acct"`
	Barcode     string     `json:"barcode"`
	Ddate       *time.Time `json:"ddate"`
	Cdate       *time.Time `json:"cdate"`
	Udate       *time.Time `json:"udate"`
	Dedate      *time.Time `json:"dedate"`
	Deathdate   *time.Time `json:"deathdate"`
	Code        string     `json:"code"`
	Cr1Id       string     `json:"cr1id"`
	Cr1Name     string     `json:"cr1name"`
	Cr2Id       string     `json:"cr2id"`
	Cr3Id       string     `json:"cr3id"`
	Cr4Id       string     `json:"cr4id"`
	Cr4Name     string     `json:"cr4name"`
	CrBrand     string     `json:"crbrand"`
	CrBrandName string     `json:"crbrandname"`
	CrMark      string     `json:"crmark"`
	CrMarkName  string     `json:"crmarkname"`
	Dcode       string     `json:"dcode"`
	Dname       string     `json:"dname"`
	Mdocno      string     `json:"mdocno"`
	Measid      string     `json:"measid"`
	Mname       string     `json:"mname"`
	Price       string     `json:"price"`
	Pricesum    string     `json:"pricesum"`
	Qty         string     `json:"qty"`
	Rid         string     `json:"rid"`
	Usize       string     `json:"usize"`
	Zno         string     `json:"zno"`
	State       uint8      `json:"state"` // New field to indicate "new", "updated", or "unchanged"
	Pkgno       string     `json:"pkgno"`
	Pkgname     string     `json:"pkgname"`
	Pkgdate     string     `json:"pkgdate"`
	Techdate    string     `json:"techdate"`
	Techurl     string     `json:"techurl"`
	Plandate    string     `json:"plandate"`
	Planurl     string     `json:"planurl"`
	Key         string     `json:"key"`
	IsArrived   bool       `json:"isArrived"`
	Tailbar     string     `json:"tailbar"`
}

type PostBasketItems struct {
	beego.Controller
}

func (c *PostBasketItems) PostBasketItems() {
	var item BasketItemPost

	err := json.Unmarshal(c.Ctx.Input.RequestBody, &item)
	if err != nil {
		fmt.Println("❌ JSON unmarshal error:", err)
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "Invalid JSON payload"}
		c.ServeJSON()
		return
	}

	item.Key = ""

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
		INSERT INTO [Tender].[dbo].[BasketItems] (
			BasketId, acct, barcode, ddate, cdate, udate, dedate,
			code, cr1id, cr1name, cr2id, cr3id, cr4id, cr4name,
			crbrand, crbrandname, crmark, crmarkname,
			dcode, dname, mdocno, measid, mname,
			price, pricesum, qty, rid, usize, zno, state,pkgno,pkgname,techdate,techurl,plandate,planurl,pkgdate,[key],isArrived,Tailbar,deathdate,IsReturned
		)
		VALUES (
			@p1, @p2, @p3, @p4, @p5, @p6, @p7,
			@p8, @p9, @p10, @p11, @p12, @p13, @p14,
			@p15, @p16, @p17, @p18,
			@p19, @p20, @p21, @p22, @p23,
			@p24, @p25, @p26, @p27, @p28, @p29,@p30,@p31,@p32,@p33,@p34,@p35,@p36,@p37,@p38,@p39,@p40,@p41,CAST(0 AS BIT)
		)
	`
	if config.Env == "prod" {
		query = `
		INSERT INTO [Tender].[logtender].[BasketItems] (
			BasketId, acct, barcode, ddate, cdate, udate, dedate,
			code, cr1id, cr1name, cr2id, cr3id, cr4id, cr4name,
			crbrand, crbrandname, crmark, crmarkname,
			dcode, dname, mdocno, measid, mname,
			price, pricesum, qty, rid, usize, zno, state,pkgno,pkgname,techdate,techurl,plandate,planurl,pkgdate,[key],isArrived,Tailbar,deathdate,IsReturned
		)
		VALUES (
			@p1, @p2, @p3, @p4, @p5, @p6, @p7,
			@p8, @p9, @p10, @p11, @p12, @p13, @p14,
			@p15, @p16, @p17, @p18,
			@p19, @p20, @p21, @p22, @p23,
			@p24, @p25, @p26, @p27, @p28, @p29,@p30,@p31,@p32,@p33,@p34,@p35,@p36,@p37,@p38,@p39,@p40,@p41,CAST(0 AS BIT)
		)
	`
	}

	tx, err := db.Begin()
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to start transaction")
		return
	}
	defer tx.Rollback()

	schema := "[Tender].[dbo]"
	if config.Env == "prod" {
		schema = "[Tender].[logtender]"
	}
	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID == 0 {
		c.CustomAbort(http.StatusUnauthorized, "Invalid user session")
		return
	}
	var ownedBasketCount int
	err = tx.QueryRow(`
		SELECT COUNT(1)
		FROM `+schema+`.[Basket]
		WHERE BasketId = @p1 AND UserId = @p2
	`, item.BasketId, claims.UserID).Scan(&ownedBasketCount)
	if err != nil || ownedBasketCount != 1 {
		c.CustomAbort(http.StatusForbidden, "Basket does not belong to authenticated user")
		return
	}
	var duplicateCount int
	var sameUserDuplicateCount int
	err = tx.QueryRow(`
		SELECT
			COUNT(1),
			ISNULL(SUM(CASE WHEN b.UserId = @p11 THEN 1 ELSE 0 END), 0)
		FROM `+schema+`.[BasketItems] bi WITH (UPDLOCK, HOLDLOCK)
		INNER JOIN `+schema+`.[Basket] b ON b.BasketId = bi.BasketId
		WHERE ISNULL(LTRIM(RTRIM(bi.code)), '') = @p1
			AND ISNULL(LTRIM(RTRIM(bi.dname)), '') = @p2
			AND ISNULL(LTRIM(RTRIM(bi.dcode)), '') = @p3
			AND ISNULL(LTRIM(RTRIM(bi.price)), '') = LTRIM(RTRIM(@p4))
			AND ISNULL(LTRIM(RTRIM(bi.qty)), '') = LTRIM(RTRIM(@p5))
			AND ISNULL(LTRIM(RTRIM(bi.mdocno)), '') = @p6
			AND ISNULL(LTRIM(RTRIM(bi.pkgno)), '') = @p7
			AND ISNULL(LTRIM(RTRIM(bi.pkgdate)), '') = @p8
			AND ISNULL(LTRIM(RTRIM(bi.rid)), '') = @p9
			AND ISNULL(LTRIM(RTRIM(bi.zno)), '') = @p10
			AND ISNULL(bi.IsReturned, 0) = 0
	`, item.Code, item.Dname, item.Dcode, item.Price, item.Qty, item.Mdocno, item.Pkgno, item.Pkgdate, item.Rid, item.Zno, claims.UserID).Scan(&duplicateCount, &sameUserDuplicateCount)
	if err != nil {
		beego.Error("Basket item duplicate validation failed:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{
			"error":   "Failed to validate basket item",
			"details": err.Error(),
		}
		c.ServeJSON()
		return
	}
	if duplicateCount > 0 {
		conflictMessage := "Order is already claimed by another user"
		if sameUserDuplicateCount > 0 {
			conflictMessage = "Order is already in your basket"
		}
		beego.Warn(
			"PostBasketItems conflict: matching item already exists",
			"basket_id:", item.BasketId,
			"user_id:", claims.UserID,
			"schema:", schema,
			"duplicate_count:", duplicateCount,
			"code:", item.Code,
			"dname:", item.Dname,
			"dcode:", item.Dcode,
			"price:", item.Price,
			"qty:", item.Qty,
			"mdocno:", item.Mdocno,
			"pkgno:", item.Pkgno,
			"pkgdate:", item.Pkgdate,
			"rid:", item.Rid,
			"zno:", item.Zno,
		)
		c.Ctx.Output.SetStatus(http.StatusConflict)
		c.Data["json"] = map[string]interface{}{
			"error":           conflictMessage,
			"duplicate_count": duplicateCount,
			"same_user_count": sameUserDuplicateCount,
		}
		c.ServeJSON()
		return
	}

	_, err = tx.Exec(query,
		item.BasketId, item.Acct, item.Barcode, item.Ddate, item.Cdate, item.Udate, item.Dedate,
		item.Code, item.Cr1Id, item.Cr1Name, item.Cr2Id, item.Cr3Id, item.Cr4Id, item.Cr4Name,
		item.CrBrand, item.CrBrandName, item.CrMark, item.CrMarkName,
		item.Dcode, item.Dname, item.Mdocno, item.Measid, item.Mname,
		item.Price, item.Pricesum, item.Qty, item.Rid, item.Usize, item.Zno, item.State, item.Pkgno,
		item.Pkgname, item.Techdate, item.Techurl, item.Plandate, item.Planurl, item.Pkgdate, item.Key, item.IsArrived, item.Tailbar, item.Deathdate,
	)

	if err != nil {
		fmt.Println("❌ Insert error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{
			"error":   "Failed to insert basket item",
			"details": err.Error(),
		}
		c.ServeJSON()
		return
	}
	if err := tx.Commit(); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to commit basket item")
		return
	}

	c.Data["json"] = map[string]string{"message": "Basket item added successfully"}
	c.ServeJSON()
}
