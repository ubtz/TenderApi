package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"fmt"
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
	fmt.Println("📥 PostBasketItems endpoint hit")
	// Log the raw JSON body
	fmt.Println("📦 Raw request body:")
	fmt.Println(string(c.Ctx.Input.RequestBody))
	var item BasketItemPost

	err := json.Unmarshal(c.Ctx.Input.RequestBody, &item)
	if err != nil {
		fmt.Println("❌ JSON unmarshal error:", err)
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "Invalid JSON payload"}
		c.ServeJSON()
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
		INSERT INTO [Tender].[dbo].[BasketItems] (
			BasketId, acct, barcode, ddate, cdate, udate, dedate,
			code, cr1id, cr1name, cr2id, cr3id, cr4id, cr4name,
			crbrand, crbrandname, crmark, crmarkname,
			dcode, dname, mdocno, measid, mname,
			price, pricesum, qty, rid, usize, zno, state,pkgno,techdate,techurl,plandate,planurl,pkgdate,[key],isArrived,Tailbar
		)
		VALUES (
			@p1, @p2, @p3, @p4, @p5, @p6, @p7,
			@p8, @p9, @p10, @p11, @p12, @p13, @p14,
			@p15, @p16, @p17, @p18,
			@p19, @p20, @p21, @p22, @p23,
			@p24, @p25, @p26, @p27, @p28, @p29,@p30,@p31,@p32,@p33,@p34,@p35,@p36,@p37,@p38,@p39
		)
	`
	if config.Env == "prod" {
		query = `
		INSERT INTO [Tender].[logtender].[BasketItems] (
			BasketId, acct, barcode, ddate, cdate, udate, dedate,
			code, cr1id, cr1name, cr2id, cr3id, cr4id, cr4name,
			crbrand, crbrandname, crmark, crmarkname,
			dcode, dname, mdocno, measid, mname,
			price, pricesum, qty, rid, usize, zno, state,pkgno,techdate,techurl,plandate,planurl,pkgdate,[key],isArrived,Tailbar
		)
		VALUES (
			@p1, @p2, @p3, @p4, @p5, @p6, @p7,
			@p8, @p9, @p10, @p11, @p12, @p13, @p14,
			@p15, @p16, @p17, @p18,
			@p19, @p20, @p21, @p22, @p23,
			@p24, @p25, @p26, @p27, @p28, @p29,@p30,@p31,@p32,@p33,@p34,@p35,@p36,@p37,@p38,@p39
		)
	`
	}
	_, err = db.Exec(query,
		item.BasketId, item.Acct, item.Barcode, item.Ddate, item.Cdate, item.Udate, item.Dedate,
		item.Code, item.Cr1Id, item.Cr1Name, item.Cr2Id, item.Cr3Id, item.Cr4Id, item.Cr4Name,
		item.CrBrand, item.CrBrandName, item.CrMark, item.CrMarkName,
		item.Dcode, item.Dname, item.Mdocno, item.Measid, item.Mname,
		item.Price, item.Pricesum, item.Qty, item.Rid, item.Usize, item.Zno, item.State, item.Pkgno,
		item.Techdate, item.Techurl, item.Plandate, item.Planurl, item.Pkgdate, item.Key, item.IsArrived, item.Tailbar,
	)

	if err != nil {
		fmt.Println("❌ Insert error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to insert basket item"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]string{"message": "Basket item added successfully"}
	c.ServeJSON()
}
