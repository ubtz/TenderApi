package controllers

import (
	config "TenderApi/conf"
	"log"

	"github.com/astaxie/beego"
	_ "github.com/denisenkom/go-mssqldb"
)

type GetBranchStatistics struct {
	beego.Controller
}

type BranchStatistics struct {
	BranchID  int     `json:"branchId"`
	Year      int     `json:"year"`
	Month     int     `json:"month"`
	Service   string  `json:"service"`
	ShortName string  `json:"shortName"`
	Orders    int     `json:"orders"`
	Qty       float64 `json:"qty"`
	Amount    float64 `json:"amount"`
}

func (c *GetBranchStatistics) GetBranchStatistics() {

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
		SELECT
		s.branch_id,
		s.year,
		s.month,
		b.service,
		b.shortName,
		SUM(s.total_orders) AS orders,
		SUM(s.total_qty) AS qty,
		SUM(s.total_amount) AS amount
	FROM [Tender].[dbo].[branch_statistics] s
	JOIN [Tender].[dbo].[branch] b ON s.branch_id = b.Id
	WHERE CAST(s.created_at AS DATE) = (
		SELECT MAX(CAST(created_at AS DATE))
		FROM [Tender].[dbo].[branch_statistics]
	)
	GROUP BY
		s.branch_id,
		s.year,
		s.month,
		b.service,
		b.shortName
	ORDER BY s.year DESC, s.month DESC
	`

	if config.Env == "prod" {
		query = `
		SELECT
		s.branch_id,
		s.year,
		s.month,
		b.service,
		b.shortName,
		SUM(s.total_orders) AS orders,
		SUM(s.total_qty) AS qty,
		SUM(s.total_amount) AS amount
	FROM [Tender].[logtender].[branch_statistics] s
	JOIN [Tender].[logtender].[branch] b ON s.branch_id = b.Id
	WHERE CAST(s.created_at AS DATE) = (
		SELECT MAX(CAST(created_at AS DATE))
		FROM [Tender].[logtender].[branch_statistics]
	)
	GROUP BY
		s.branch_id,
		s.year,
		s.month,
		b.service,
		b.shortName
	ORDER BY s.year DESC, s.month DESC
		`
	}

	rows, err := db.Query(query)

	if err != nil {
		log.Println("Query error:", err)

		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{
			"error": "Query failed",
		}

		c.ServeJSON()
		return
	}

	defer rows.Close()

	var stats []BranchStatistics

	for rows.Next() {

		var s BranchStatistics

		err := rows.Scan(
			&s.BranchID,
			&s.Year,
			&s.Month,
			&s.Service,
			&s.ShortName,
			&s.Orders,
			&s.Qty,
			&s.Amount,
		)

		if err != nil {
			log.Println("Row scan error:", err)
			continue
		}

		stats = append(stats, s)
	}

	if err := rows.Err(); err != nil {

		log.Println("Row iteration error:", err)

		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{
			"error": "Failed to read rows",
		}

		c.ServeJSON()
		return
	}

	c.Data["json"] = stats
	c.ServeJSON()
}
