package controllers

import (
	config "TenderApi/conf"
	"fmt"
	"strings"

	"github.com/astaxie/beego"
)

type GetStatistic struct {
	beego.Controller
}

func (c *GetStatistic) GetStatistic() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// 🧩 SQL base query — uses [Tender].[dbo] by default
	query := `
	SET NOCOUNT ON;
SET ANSI_WARNINGS OFF;

WITH BasketStats AS (
	SELECT
		-- ✅ Count every record as an order (no DISTINCT)
		COUNT(*) AS TotalRequestCount,
		SUM(
			CASE 
				-- ✅ Handle scientific notation (e.g., 1.23e+06)
				WHEN TRY_CAST(REPLACE(pricesum, ',', '') AS FLOAT) IS NOT NULL 
				THEN TRY_CAST(REPLACE(pricesum, ',', '') AS FLOAT)
				ELSE 0
			END
		) AS TotalRequestAmount
	FROM [Tender].[dbo].[BasketItems]
	WHERE 
		CASE 
			WHEN ISDATE(REPLACE(pkgdate, '/', '-')) = 1 
			THEN CONVERT(datetime, REPLACE(pkgdate, '/', '-')) 
			ELSE NULL 
		END BETWEEN '2025-01-01' AND '2025-12-31'
),
TenderStats AS (
	SELECT
		COUNT(DISTINCT [Тендерийн_дугаар]) AS TotalTenders,

		SUM(
			CASE 
				WHEN TRY_CAST(REPLACE([Батлагдсан_төсөвт_өртөг], ',', '') AS FLOAT) IS NOT NULL 
				THEN TRY_CAST(REPLACE([Батлагдсан_төсөвт_өртөг], ',', '') AS FLOAT)
				ELSE 0 
			END
		) AS TotalBudget,

		SUM(CASE WHEN CAST([Тендер_амжилттай_болсон_эсэх] AS NVARCHAR(255)) = N'Тийм' THEN 1 ELSE 0 END) AS SuccessfulTenders,

		SUM(
			CASE 
				WHEN CAST([Тендер_амжилттай_болсон_эсэх] AS NVARCHAR(255)) = N'Тийм'
					AND TRY_CAST(REPLACE([Батлагдсан_төсөвт_өртөг], ',', '') AS FLOAT) IS NOT NULL
				THEN TRY_CAST(REPLACE([Батлагдсан_төсөвт_өртөг], ',', '') AS FLOAT)
				ELSE 0 
			END
		) AS SuccessfulBudget,

		SUM(CASE WHEN [Түтгэлзүүлсэн_огноо] IS NOT NULL THEN 1 ELSE 0 END) AS SuspendedTenders
	FROM [Tender].[dbo].[Tender]
	WHERE 
		CASE 
			WHEN ISDATE([Урилгийн_огноо]) = 1 
			THEN CONVERT(datetime, [Урилгийн_огноо]) 
			ELSE NULL 
		END BETWEEN '2025-01-01' AND '2025-12-31'
),
GereeStats AS (
	SELECT
		COUNT([GereeId]) AS TotalContracts,
		SUM(
			CASE 
				WHEN TRY_CAST(REPLACE([Гэрээний_дүн], ',', '') AS FLOAT) IS NOT NULL 
				THEN TRY_CAST(REPLACE([Гэрээний_дүн], ',', '') AS FLOAT)
				ELSE 0 
			END
		) AS TotalContractValue
	FROM [Tender].[dbo].[Geree]
	WHERE 
		CASE 
			WHEN ISDATE([Гэрээ_байгуулсан_огноо]) = 1 
			THEN CONVERT(datetime, [Гэрээ_байгуулсан_огноо]) 
			ELSE NULL 
		END BETWEEN '2025-01-01' AND '2025-12-31'
)
SELECT
	b.TotalRequestCount     AS TotalOrders,
	b.TotalRequestAmount    AS TotalOrderValue,
	t.TotalTenders,
	t.TotalBudget,
	t.SuccessfulTenders,
	t.SuccessfulBudget,
	t.SuspendedTenders,
	g.TotalContracts,
	g.TotalContractValue
FROM BasketStats b
CROSS JOIN TenderStats t
CROSS JOIN GereeStats g;

`

	// 🧠 Replace schema if prod
	if config.Env == "prod" {
		query = strings.ReplaceAll(query, "[Tender].[dbo]", "[Tender].[logtender]")
	}

	// ✅ Execute query
	row := db.QueryRow(query)

	type Statistic struct {
		TotalOrders        int
		TotalOrderValue    float64
		TotalTenders       int
		TotalBudget        float64
		SuccessfulTenders  int
		SuccessfulBudget   float64
		SuspendedTenders   int
		TotalContracts     int
		TotalContractValue float64
	}

	var s Statistic
	err := row.Scan(
		&s.TotalOrders,
		&s.TotalOrderValue,
		&s.TotalTenders,
		&s.TotalBudget,
		&s.SuccessfulTenders,
		&s.SuccessfulBudget,
		&s.SuspendedTenders,
		&s.TotalContracts,
		&s.TotalContractValue,
	)
	if err != nil {
		fmt.Println("❌ Statistic query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to get statistics"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = s
	c.ServeJSON()
}
