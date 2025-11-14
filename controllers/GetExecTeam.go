package controllers

import (
	"database/sql"
	"fmt"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type GetExecTeam struct {
	beego.Controller
}

// ✅ OUTPUT STRUCT — DB-ээс буцах бүтэц
type ExecTeam struct {
	PlanRootNumber string `json:"planRootNumber"`
	Batlah         string `json:"Батлах"`
	Zuvshuursun    string `json:"Зөвшөөрсөн"`
	Guitsetgesen   string `json:"Гүйцэтгэсэн"`
	UserId         int    `json:"userId"` // ✅ Added
}

// ✅ GET /execTeam
func (c *GetExecTeam) GetExecTeam() {
	fmt.Println("📥 GetExecTeam endpoint hit")

	// DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// ✅ Include UserId in query
	query := `
		SELECT TOP (1000) [PlanRootNumber], [Батлах], [Зөвшөөрсөн], [Гүйцэтгэсэн], [UserId]
		FROM [Tender].[dbo].[ExecTeam]
		ORDER BY PlanRootNumber DESC
	`
	if config.Env == "prod" {
		query = `
			SELECT TOP (1000) [PlanRootNumber], [Батлах], [Зөвшөөрсөн], [Гүйцэтгэсэн], [UserId]
			FROM [Tender].[logtender].[ExecTeam]
			ORDER BY PlanRootNumber DESC
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("❌ Failed to query ExecTeam:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to fetch ExecTeam"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var results []ExecTeam

	for rows.Next() {
		var et ExecTeam
		if err := rows.Scan(
			&et.PlanRootNumber,
			&et.Batlah,
			&et.Zuvshuursun,
			&et.Guitsetgesen,
			&et.UserId, // ✅ Added
		); err != nil {
			if err == sql.ErrNoRows {
				continue
			}
			fmt.Println("❌ Scan error:", err)
			continue
		}
		results = append(results, et)
	}

	// ✅ Return JSON
	c.Data["json"] = results
	c.ServeJSON()
}
