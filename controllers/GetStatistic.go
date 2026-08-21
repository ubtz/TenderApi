package controllers

import (
	config "TenderApi/conf"
	"fmt"

	"github.com/astaxie/beego"
)

type GetStatistic struct {
	beego.Controller
}

type Statistic struct {
	Dcode              string  `json:"dcode"`
	TotalOrders        int     `json:"TotalOrders"`
	TotalOrderValue    float64 `json:"TotalOrderValue"`
	TotalTenders       int     `json:"TotalTenders"`
	TotalBudget        float64 `json:"TotalBudget"`
	SuccessfulTenders  int     `json:"SuccessfulTenders"`
	SuccessfulBudget   float64 `json:"SuccessfulBudget"`
	SuspendedTenders   int     `json:"SuspendedTenders"`
	TotalContracts     int     `json:"TotalContracts"`
	TotalContractValue float64 `json:"TotalContractValue"`
}

type StatisticResponse struct {
	Rows    []Statistic `json:"rows"`
	Summary Statistic   `json:"summary"`
}

func (c *GetStatistic) GetStatistic() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	baseSchema := "[Tender].[dbo]"
	if config.Env == "prod" {
		baseSchema = "[Tender].[logtender]"
	}

	rowsQuery := `
SET NOCOUNT ON;
SET ANSI_WARNINGS OFF;

WITH BranchMap AS (
	SELECT
		CAST(id AS NVARCHAR(255)) AS dcode,
		LTRIM(RTRIM(CAST(shortName AS NVARCHAR(255)))) AS shortName
	FROM ` + baseSchema + `.[branch]
),
BasketStats AS (
	SELECT
		ISNULL(CAST(bi.dcode AS NVARCHAR(255)), '') AS dcode,
		COUNT(*) AS TotalOrders,
		SUM(
			CASE
				WHEN ISNUMERIC(REPLACE(CAST(bi.pricesum AS NVARCHAR(MAX)), ',', '')) = 1
				THEN CAST(REPLACE(CAST(bi.pricesum AS NVARCHAR(MAX)), ',', '') AS FLOAT)
				ELSE 0
			END
		) AS TotalOrderValue
	FROM ` + baseSchema + `.[BasketItems] bi
	WHERE ISNULL(bi.IsReturned, 0) = 0
	GROUP BY ISNULL(CAST(bi.dcode AS NVARCHAR(255)), '')
),
TenderExpanded AS (
	SELECT DISTINCT
		t.TenderId,
		bm.dcode,
		CASE
			WHEN ISNUMERIC(REPLACE(CAST(t.[Батлагдсан_төсөвт_өртөг] AS NVARCHAR(MAX)), ',', '')) = 1
			THEN CAST(REPLACE(CAST(t.[Батлагдсан_төсөвт_өртөг] AS NVARCHAR(MAX)), ',', '') AS FLOAT)
			ELSE 0
		END AS BudgetValue,
		CASE
			WHEN CAST(t.[Тендер_амжилттай_болсон_эсэх] AS NVARCHAR(255)) IN (N'Тийм', N'true', N'TRUE', N'True', N'1')
			THEN 1 ELSE 0
		END AS IsSuccessful,
		CASE
			WHEN t.[Түтгэлзүүлсэн_огноо] IS NOT NULL
				AND ISDATE(CAST(t.[Түтгэлзүүлсэн_огноо] AS NVARCHAR(30))) = 1
				AND CAST(CAST(t.[Түтгэлзүүлсэн_огноо] AS NVARCHAR(30)) AS DATETIME) <> CAST('19000101' AS DATETIME)
			THEN 1 ELSE 0
		END AS IsSuspended
	FROM ` + baseSchema + `.[Tender] t
	INNER JOIN BranchMap bm
		ON CHARINDEX(
			(',' + bm.shortName + ',') COLLATE DATABASE_DEFAULT,
			(',' + REPLACE(CAST(t.[Organization] AS NVARCHAR(MAX)), ', ', ',') + ',') COLLATE DATABASE_DEFAULT
		) > 0
	WHERE t.RootTenderId IS NULL
	  AND ISNULL(t.IsDeleted, 0) = 0
),
TenderStats AS (
	SELECT
		te.dcode,
		COUNT(DISTINCT te.TenderId) AS TotalTenders,
		SUM(te.BudgetValue) AS TotalBudget,
		SUM(CASE WHEN te.IsSuccessful = 1 THEN 1 ELSE 0 END) AS SuccessfulTenders,
		SUM(CASE WHEN te.IsSuccessful = 1 THEN te.BudgetValue ELSE 0 END) AS SuccessfulBudget,
		SUM(CASE WHEN te.IsSuspended = 1 THEN 1 ELSE 0 END) AS SuspendedTenders
	FROM TenderExpanded te
	GROUP BY te.dcode
),
GereeStats AS (
	SELECT
		COUNT(g.[GereeId]) AS TotalContracts,
		SUM(
			CASE
				WHEN ISNUMERIC(REPLACE(CAST(g.[Гэрээний_дүн] AS NVARCHAR(MAX)), ',', '')) = 1
				THEN CAST(REPLACE(CAST(g.[Гэрээний_дүн] AS NVARCHAR(MAX)), ',', '') AS FLOAT)
				ELSE 0
			END
		) AS TotalContractValue
	FROM ` + baseSchema + `.[Geree] g
),
AllDcodes AS (
	SELECT dcode FROM BasketStats
	UNION
	SELECT dcode FROM TenderStats
)
SELECT
	a.dcode,
	ISNULL(b.TotalOrders, 0) AS TotalOrders,
	ISNULL(b.TotalOrderValue, 0) AS TotalOrderValue,
	ISNULL(t.TotalTenders, 0) AS TotalTenders,
	ISNULL(t.TotalBudget, 0) AS TotalBudget,
	ISNULL(t.SuccessfulTenders, 0) AS SuccessfulTenders,
	ISNULL(t.SuccessfulBudget, 0) AS SuccessfulBudget,
	ISNULL(t.SuspendedTenders, 0) AS SuspendedTenders,
	ISNULL(g.TotalContracts, 0) AS TotalContracts,
	ISNULL(g.TotalContractValue, 0) AS TotalContractValue
FROM AllDcodes a
LEFT JOIN BasketStats b ON a.dcode = b.dcode
LEFT JOIN TenderStats t ON a.dcode = t.dcode
CROSS JOIN GereeStats g
ORDER BY a.dcode;
`

	summaryQuery := `
SET NOCOUNT ON;
SET ANSI_WARNINGS OFF;

SELECT
	'' AS dcode,
	ISNULL(o.TotalOrders, 0) AS TotalOrders,
	ISNULL(o.TotalOrderValue, 0) AS TotalOrderValue,
	ISNULL(t.TotalTenders, 0) AS TotalTenders,
	ISNULL(t.TotalBudget, 0) AS TotalBudget,
	ISNULL(t.SuccessfulTenders, 0) AS SuccessfulTenders,
	ISNULL(t.SuccessfulBudget, 0) AS SuccessfulBudget,
	ISNULL(t.SuspendedTenders, 0) AS SuspendedTenders,
	ISNULL(g.TotalContracts, 0) AS TotalContracts,
	ISNULL(g.TotalContractValue, 0) AS TotalContractValue
FROM (
	SELECT
		COUNT(*) AS TotalOrders,
		SUM(
			CASE
				WHEN ISNUMERIC(REPLACE(CAST(pricesum AS NVARCHAR(MAX)), ',', '')) = 1
				THEN CAST(REPLACE(CAST(pricesum AS NVARCHAR(MAX)), ',', '') AS FLOAT)
				ELSE 0
			END
		) AS TotalOrderValue
	FROM ` + baseSchema + `.[BasketItems]
	WHERE ISNULL(IsReturned, 0) = 0
) o
CROSS JOIN (
	SELECT
		COUNT(TenderId) AS TotalTenders,
		SUM(
			CASE
				WHEN ISNUMERIC(REPLACE(CAST([Батлагдсан_төсөвт_өртөг] AS NVARCHAR(MAX)), ',', '')) = 1
				THEN CAST(REPLACE(CAST([Батлагдсан_төсөвт_өртөг] AS NVARCHAR(MAX)), ',', '') AS FLOAT)
				ELSE 0
			END
		) AS TotalBudget,
		SUM(
			CASE
				WHEN CAST([Тендер_амжилттай_болсон_эсэх] AS NVARCHAR(255)) IN (N'Тийм', N'true', N'TRUE', N'True', N'1')
				THEN 1 ELSE 0
			END
		) AS SuccessfulTenders,
		SUM(
			CASE
				WHEN CAST([Тендер_амжилттай_болсон_эсэх] AS NVARCHAR(255)) IN (N'Тийм', N'true', N'TRUE', N'True', N'1')
					AND ISNUMERIC(REPLACE(CAST([Батлагдсан_төсөвт_өртөг] AS NVARCHAR(MAX)), ',', '')) = 1
				THEN CAST(REPLACE(CAST([Батлагдсан_төсөвт_өртөг] AS NVARCHAR(MAX)), ',', '') AS FLOAT)
				ELSE 0
			END
		) AS SuccessfulBudget,
		SUM(
			CASE
				WHEN [Түтгэлзүүлсэн_огноо] IS NOT NULL
					AND ISDATE(CAST([Түтгэлзүүлсэн_огноо] AS NVARCHAR(30))) = 1
					AND CAST(CAST([Түтгэлзүүлсэн_огноо] AS NVARCHAR(30)) AS DATETIME) <> CAST('19000101' AS DATETIME)
				THEN 1 ELSE 0
			END
		) AS SuspendedTenders
	FROM ` + baseSchema + `.[Tender]
	WHERE RootTenderId IS NULL
	  AND ISNULL(IsDeleted, 0) = 0
) t
CROSS JOIN (
	SELECT
		COUNT([GereeId]) AS TotalContracts,
		SUM(
			CASE
				WHEN ISNUMERIC(REPLACE(CAST([Гэрээний_дүн] AS NVARCHAR(MAX)), ',', '')) = 1
				THEN CAST(REPLACE(CAST([Гэрээний_дүн] AS NVARCHAR(MAX)), ',', '') AS FLOAT)
				ELSE 0
			END
		) AS TotalContractValue
	FROM ` + baseSchema + `.[Geree]
) g;
`

	rows, err := db.Query(rowsQuery)
	if err != nil {
		fmt.Println("❌ Statistic rows query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to get statistics rows"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var result []Statistic
	for rows.Next() {
		var s Statistic
		err := rows.Scan(
			&s.Dcode,
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
			fmt.Println("❌ Row scan error:", err)
			c.Ctx.Output.SetStatus(500)
			c.Data["json"] = map[string]string{"error": "Failed to scan statistics rows"}
			c.ServeJSON()
			return
		}
		result = append(result, s)
	}

	if err = rows.Err(); err != nil {
		fmt.Println("❌ Rows iteration error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed while reading statistics rows"}
		c.ServeJSON()
		return
	}

	var summary Statistic
	err = db.QueryRow(summaryQuery).Scan(
		&summary.Dcode,
		&summary.TotalOrders,
		&summary.TotalOrderValue,
		&summary.TotalTenders,
		&summary.TotalBudget,
		&summary.SuccessfulTenders,
		&summary.SuccessfulBudget,
		&summary.SuspendedTenders,
		&summary.TotalContracts,
		&summary.TotalContractValue,
	)
	if err != nil {
		fmt.Println("❌ Statistic summary query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to get statistics summary"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = StatisticResponse{
		Rows:    result,
		Summary: summary,
	}
	c.ServeJSON()
}
