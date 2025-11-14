package controllers

import (
	config "TenderApi/conf"
	"log"

	"github.com/astaxie/beego"
	_ "github.com/denisenkom/go-mssqldb"
)

type GetBranch struct {
	beego.Controller
}

// Struct to hold each row of the query result
type Branch struct {
	Id        int    `json:"id"`
	Branch    string `json:"branch"`
	ShortName string `json:"shortName"`
	Service   string `json:"service"`
}

func (c *GetBranch) GetBranch() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Run the SQL query
	rows, err := db.Query(`SELECT TOP (10000) [Id], [branch], [shortName], [service] FROM [Tender].[dbo].[branch]`)
	if config.Env == "prod" {
		rows, err = db.Query(`SELECT TOP (10000) [Id], [branch], [shortName], [service] FROM [Tender].[logtender].[branch]`)
	}
	if err != nil {
		log.Println("Query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Query failed"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	// Prepare the result slice
	var branches []Branch

	for rows.Next() {
		var b Branch
		err := rows.Scan(&b.Id, &b.Branch, &b.ShortName, &b.Service)
		if err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		branches = append(branches, b)
	}

	// Handle errors from iteration
	if err := rows.Err(); err != nil {
		log.Println("Row iteration error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to read rows"}
		c.ServeJSON()
		return
	}

	// Return the result
	c.Data["json"] = branches
	c.ServeJSON()
}
