package controllers

import (
	"database/sql"
	"fmt"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type GetUnelgeeHoroo struct {
	beego.Controller
}

// OUTPUT STRUCT — DB-ээс буцах бүтэц
type UnelgeeHoroo struct {
	UnelgeeHorooId int    `json:"unelgeeHorooId"`
	TenderId       int    `json:"tenderId"`
	Darga          string `json:"darga"`
	Gishuud        string `json:"gishuud"`
	Dugaar         string `json:"dugaar"`
	Created        string `json:"created"`
}

// GET /unelgeeHoroo
func (c *GetUnelgeeHoroo) GetUnelgeeHoroo() {
	fmt.Println("📥 GetUnelgeeHoroo endpoint hit")

	// DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
		SELECT TOP (1000) [UnelgeeHorooId], [TenderId], [Darga], [Gishuud], [Dugaar], [Created]
		FROM [Tender].[dbo].[UnelgeeHoroo]
		ORDER BY UnelgeeHorooId DESC
	`
	if config.Env == "prod" {
		query = `
		SELECT TOP (1000) [UnelgeeHorooId], [TenderId], [Darga], [Gishuud], [Dugaar], [Created]
		FROM [Tender].[logtender].[UnelgeeHoroo]
		ORDER BY UnelgeeHorooId DESC
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("❌ Failed to query UnelgeeHoroo:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to fetch UnelgeeHoroo"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var results []UnelgeeHoroo

	for rows.Next() {
		var uh UnelgeeHoroo
		if err := rows.Scan(&uh.UnelgeeHorooId, &uh.TenderId, &uh.Darga, &uh.Gishuud, &uh.Dugaar, &uh.Created); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			fmt.Println("❌ Scan error:", err)
			continue
		}
		results = append(results, uh)
	}

	// return JSON
	c.Data["json"] = results
	c.ServeJSON()
}
