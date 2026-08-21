package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"strings"

	"github.com/astaxie/beego"
)

type GetOrders struct {
	beego.Controller
}

func (c *GetOrders) GetOrders() {

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
	SELECT TOP (100000)
		[id],
		[rid],
		[acct],
		[code],
		[cr1id],
		[cr2id],
		[cr3id],
		[cr4id],
		[crbrand],
		[crmark],
		[cr1name],
		[cr4name],
		[crbrandname],
		[crmarkname],
		[mdocno],
		[pkgno],
		[ddate],
		[pkgdate],
		[dedate],
		[cdate],
		[udate],
		[plandate],
		[techdate],
		[measid],
		[mname],
		[usize],
		[barcode],
		[zno],
		[qty],
		[qty_nha],
		[price],
		[pricesum],
		[dcode],
		[dname],
		[plan],
		[tech],
		[planurl],
		[techurl],
		[man]
	FROM [Tender].[dbo].[orders]
	`

	// ✅ PROD SWITCH
	if config.Env == "prod" {
		query = strings.ReplaceAll(query, "[Tender].[dbo]", "[Tender].[logtender]")
	}

	rows, err := db.Query(query)
	if err != nil {
		beego.Error("query error:", err)

		c.Data["json"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	results := []map[string]interface{}{}

	for rows.Next() {

		var (
			id, man                    int
			rid, acct, code            string
			cr1id, cr2id, cr3id, cr4id string
			crbrand, crmark            string
			cr1name, cr4name           string
			crbrandname, crmarkname    string
			mdocno, pkgno              string
			measid, mname, usize       string
			barcode, zno               string
			dcode, dname               string
			plan, tech                 string
			planurl, techurl           string

			ddate, pkgdate, dedate interface{}
			cdate, udate           interface{}
			plandate, techdate     interface{}

			qty, qty_nha, price, pricesum sql.NullFloat64
		)

		err := rows.Scan(
			&id,
			&rid,
			&acct,
			&code,
			&cr1id,
			&cr2id,
			&cr3id,
			&cr4id,
			&crbrand,
			&crmark,
			&cr1name,
			&cr4name,
			&crbrandname,
			&crmarkname,
			&mdocno,
			&pkgno,
			&ddate,
			&pkgdate,
			&dedate,
			&cdate,
			&udate,
			&plandate,
			&techdate,
			&measid,
			&mname,
			&usize,
			&barcode,
			&zno,
			&qty,
			&qty_nha,
			&price,
			&pricesum,
			&dcode,
			&dname,
			&plan,
			&tech,
			&planurl,
			&techurl,
			&man,
		)

		if err != nil {
			beego.Error("scan error:", err)
			continue
		}

		getFloat := func(f sql.NullFloat64) interface{} {
			if f.Valid {
				return f.Float64
			}
			return nil
		}

		row := map[string]interface{}{
			"id":          id,
			"rid":         rid,
			"acct":        acct,
			"code":        code,
			"cr1id":       cr1id,
			"cr2id":       cr2id,
			"cr3id":       cr3id,
			"cr4id":       cr4id,
			"crbrand":     crbrand,
			"crmark":      crmark,
			"cr1name":     cr1name,
			"cr4name":     cr4name,
			"crbrandname": crbrandname,
			"crmarkname":  crmarkname,
			"mdocno":      mdocno,
			"pkgno":       pkgno,
			"ddate":       ddate,
			"pkgdate":     pkgdate,
			"dedate":      dedate,
			"cdate":       cdate,
			"udate":       udate,
			"plandate":    plandate,
			"techdate":    techdate,
			"measid":      measid,
			"mname":       mname,
			"usize":       usize,
			"barcode":     barcode,
			"zno":         zno,
			"qty":         getFloat(qty),
			"qty_nha":     getFloat(qty_nha),
			"price":       getFloat(price),
			"pricesum":    getFloat(pricesum),
			"dcode":       dcode,
			"dname":       dname,
			"plan":        plan,
			"tech":        tech,
			"planurl":     planurl,
			"techurl":     techurl,
			"man":         man,
		}

		results = append(results, row)
	}

	c.Data["json"] = map[string]interface{}{
		"success": true,
		"data":    results,
	}

	c.ServeJSON()
}
