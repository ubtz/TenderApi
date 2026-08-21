package controllers

import (
	config "TenderApi/conf"
	"log"

	"github.com/astaxie/beego"
	_ "github.com/denisenkom/go-mssqldb"
)

type GetItems struct {
	beego.Controller
}

type BasketItemAll struct {
	BasketItemId  int     `json:"basketItemId"`
	BasketId      int     `json:"basketId"`
	OwnerUsername string  `json:"ownerUsername"`
	OwnerOvog     string  `json:"ownerOvog"`
	OwnerNer      string  `json:"ownerNer"`
	UserId        int     `json:"userId"` // ✅ added
	AddedAt       string  `json:"addedAt"`
	Acct          string  `json:"acct"`
	Barcode       string  `json:"barcode"`
	Ddate         string  `json:"ddate"`
	Cdate         string  `json:"cdate"`
	Udate         string  `json:"udate"`
	Dedate        string  `json:"dedate"`
	Deathdate     string  `json:"deathdate"`
	Code          string  `json:"code"`
	Cr4name       string  `json:"cr4name"`
	Crbrand       string  `json:"crbrand"`
	Crbrandname   string  `json:"crbrandname"`
	Crmark        string  `json:"crmark"`
	Crmarkname    string  `json:"crmarkname"`
	Dcode         string  `json:"dcode"`
	Dname         string  `json:"dname"`
	Mdocno        string  `json:"mdocno"`
	Measid        string  `json:"measid"`
	Mname         string  `json:"mname"`
	Price         float64 `json:"price"`
	Pricesum      float64 `json:"pricesum"`
	Qty           float64 `json:"qty"`
	Rid           string  `json:"rid"`
	Usize         string  `json:"usize"`
	Zno           string  `json:"zno"`
	Planurl       string  `json:"planurl"`
	Techurl       string  `json:"techurl"`
	Pkgno         string  `json:"pkgno"`
	Pkgname       string  `json:"pkgname"`
	Pkgdate       string  `json:"pkgdate"`
}

func (c *GetItems) GetItems() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
		SELECT TOP (100000)
			bi.[BasketItemId],
			bi.[BasketId],
			b.[UserId],               -- ✅ Join Basket to get UserId
			ISNULL(u.[Username], ''),
			ISNULL(u.[Ovog], ''),
			ISNULL(u.[Ner], ''),
			bi.[AddedAt],
			bi.[acct],
			bi.[barcode],
			bi.[ddate],
			bi.[cdate],
			bi.[udate],
			bi.[dedate],
			ISNULL(CONVERT(VARCHAR(30), bi.[deathdate], 126), ''),
			bi.[code],
			bi.[cr4name],
			bi.[crbrand],
			bi.[crbrandname],
			bi.[crmark],
			bi.[crmarkname],
			bi.[dcode],
			bi.[dname],
			bi.[mdocno],
			bi.[measid],
			bi.[mname],
			bi.[price],
			bi.[pricesum],
			bi.[qty],
			bi.[rid],
			bi.[usize],
			bi.[zno],
			ISNULL(bi.[planurl], ''),
			ISNULL(bi.[techurl], ''),
			bi.[pkgno],
			ISNULL(bi.[pkgname], ''),
			bi.[pkgdate]
		FROM [Tender].[dbo].[BasketItems] AS bi
		INNER JOIN [Tender].[dbo].[Basket] AS b
			ON bi.BasketId = b.BasketId
		LEFT JOIN [Tender].[dbo].[Users] AS u
			ON b.UserId = u.Id
		WHERE ISNULL(bi.IsReturned, 0) = 0
	`

	if config.Env == "prod" {
		query = `
			SELECT TOP (100000)
				bi.[BasketItemId],
				bi.[BasketId],
				b.[UserId],
				ISNULL(u.[Username], ''),
				ISNULL(u.[Ovog], ''),
				ISNULL(u.[Ner], ''),
				bi.[AddedAt],
				bi.[acct],
				bi.[barcode],
				bi.[ddate],
				bi.[cdate],
				bi.[udate],
				bi.[dedate],
				ISNULL(CONVERT(VARCHAR(30), bi.[deathdate], 126), ''),
				bi.[code],
				bi.[cr4name],
				bi.[crbrand],
				bi.[crbrandname],
				bi.[crmark],
				bi.[crmarkname],
				bi.[dcode],
				bi.[dname],
				bi.[mdocno],
				bi.[measid],
				bi.[mname],
				bi.[price],
				bi.[pricesum],
				bi.[qty],
				bi.[rid],
				bi.[usize],
				bi.[zno],
				ISNULL(bi.[planurl], ''),
				ISNULL(bi.[techurl], ''),
				bi.[pkgno],
				ISNULL(bi.[pkgname], ''),
				bi.[pkgdate]
			FROM [Tender].[logtender].[BasketItems] AS bi
			INNER JOIN [Tender].[logtender].[Basket] AS b
				ON bi.BasketId = b.BasketId
			LEFT JOIN [Tender].[logtender].[Users] AS u
				ON b.UserId = u.Id
			WHERE ISNULL(bi.IsReturned, 0) = 0
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Println("Query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Query failed"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var items []BasketItemAll

	for rows.Next() {
		var bi BasketItemAll
		err := rows.Scan(
			&bi.BasketItemId, &bi.BasketId, &bi.UserId, // ✅ added UserId here
			&bi.OwnerUsername, &bi.OwnerOvog, &bi.OwnerNer,
			&bi.AddedAt, &bi.Acct, &bi.Barcode, &bi.Ddate, &bi.Cdate, &bi.Udate, &bi.Dedate, &bi.Deathdate, &bi.Code,
			&bi.Cr4name, &bi.Crbrand, &bi.Crbrandname, &bi.Crmark, &bi.Crmarkname,
			&bi.Dcode, &bi.Dname, &bi.Mdocno, &bi.Measid, &bi.Mname,
			&bi.Price, &bi.Pricesum, &bi.Qty, &bi.Rid, &bi.Usize, &bi.Zno,
			&bi.Planurl, &bi.Techurl, &bi.Pkgno, &bi.Pkgname, &bi.Pkgdate,
		)
		if err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		items = append(items, bi)
	}

	if err := rows.Err(); err != nil {
		log.Println("Row iteration error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to read rows"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = items
	c.ServeJSON()
}
