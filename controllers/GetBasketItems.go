package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/astaxie/beego"
)

// =======================
// Controller
// =======================

type GetBasketItems struct {
	beego.Controller
}

// =======================
// Model
// =======================

type BasketItem struct {
	BasketItemId int            `json:"basket_item_id"`
	BasketId     int            `json:"basket_id"`
	AddedAt      sql.NullString `json:"added_at"`

	Code       sql.NullString `json:"code"`
	Cr4Name    sql.NullString `json:"cr4name"`
	CrMarkName sql.NullString `json:"crmarkname"`

	DCode sql.NullString `json:"dcode"`
	DName sql.NullString `json:"dname"`

	MDocNo sql.NullString `json:"mdocno"`
	MName  sql.NullString `json:"mname"`
	USize  sql.NullString `json:"usize"`

	Qty      sql.NullInt64   `json:"qty"`
	Price    sql.NullFloat64 `json:"price"`
	PriceSum sql.NullFloat64 `json:"pricesum"`

	PkgNo     sql.NullString `json:"pkgno"`
	PkgName   sql.NullString `json:"pkgname"`
	PkgDate   sql.NullString `json:"pkgdate"`
	DeathDate sql.NullString `json:"deathdate"`

	IsArrived sql.NullBool   `json:"isArrived"`
	Tailbar   sql.NullString `json:"tailbar"`

	// ✅ GENERATED (NOT STORED)
	CurrentStepKey string `json:"current_step_key"`
}

// =======================
// GET /get/GetBasketItems
// =======================

func (c *GetBasketItems) GetBasketItems() {
	fmt.Println("📥 GetBasketItems called")

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
	SELECT
		bi.BasketItemId,
		bi.BasketId,
		bi.AddedAt,
		bi.code,
		bi.cr4name,
		bi.crmarkname,
		bi.dcode,
		bi.dname,
		bi.mdocno,
		bi.mname,
		bi.usize,
		bi.qty,
		bi.price,
		bi.pricesum,
		bi.pkgno,
		bi.pkgname,
		bi.pkgdate,
		bi.deathdate,
		bi.isArrived,
		bi.Tailbar,

		-- =========================
		-- CURRENT STEP KEY (SAME LOGIC AS GetItemSteps)
		-- =========================
		CASE
			-- 🏁 Contract (Geree) wins
			WHEN EXISTS (
				SELECT 1
				FROM [Tender].[dbo].[Geree] g
				WHERE EXISTS (
					SELECT 1
					FROM (
						SELECT LTRIM(RTRIM(x.i.value('.', 'VARCHAR(20)'))) AS basket_id
						FROM (
							SELECT CAST(
								'<i>' + REPLACE(g.Basket_Ids, ',', '</i><i>') + '</i>'
								AS XML
							) AS xmlData
						) a
						CROSS APPLY xmlData.nodes('/i') x(i)
					) s
					WHERE TRY_CAST(s.basket_id AS INT) = bi.BasketId
				)
			) THEN 'contract_signed'

			-- 🚨 Tender next
			WHEN EXISTS (
				SELECT 1
				FROM [Tender].[dbo].[Tender] t
				WHERE EXISTS (
					SELECT 1
					FROM (
						SELECT LTRIM(RTRIM(x.i.value('.', 'VARCHAR(20)'))) AS basket_id
						FROM (
							SELECT CAST(
								'<i>' + REPLACE(t.Basket_Ids, ',', '</i><i>') + '</i>'
								AS XML
							) AS xmlData
						) a
						CROSS APPLY xmlData.nodes('/i') x(i)
					) s
					WHERE TRY_CAST(s.basket_id AS INT) = bi.BasketId
				)
			) THEN 'tender_started'

			-- 📦 Item-side (descending priority)
			WHEN bi.AddedAt IS NOT NULL THEN 'planned_by_specialist'
			WHEN bi.udate IS NOT NULL THEN 'nh_assigned'
			WHEN bi.cdate IS NOT NULL THEN 'department_to_nh'
			WHEN bi.dedate IS NOT NULL THEN 'sent_to_department'
			WHEN bi.ddate IS NOT NULL THEN 'created_by_company'

			ELSE 'created_by_company'
		END AS current_step_key

	FROM [Tender].[dbo].[BasketItems] bi
	WHERE ISNULL(bi.IsReturned, 0) = 0
	`

	if config.Env == "prod" {
		query = `
		SELECT
			bi.BasketItemId,
			bi.BasketId,
			bi.AddedAt,
			bi.code,
			bi.cr4name,
			bi.crmarkname,
			bi.dcode,
			bi.dname,
			bi.mdocno,
			bi.mname,
			bi.usize,
			bi.qty,
			bi.price,
			bi.pricesum,
			bi.pkgno,
			bi.pkgname,
			bi.pkgdate,
			bi.deathdate,
			bi.isArrived,
			bi.Tailbar,

			CASE
				WHEN EXISTS (
					SELECT 1
					FROM [Tender].[logtender].[Geree] g
					WHERE EXISTS (
						SELECT 1
						FROM (
							SELECT LTRIM(RTRIM(x.i.value('.', 'VARCHAR(20)'))) AS basket_id
							FROM (
								SELECT CAST(
									'<i>' + REPLACE(g.Basket_Ids, ',', '</i><i>') + '</i>'
									AS XML
								) AS xmlData
							) a
							CROSS APPLY xmlData.nodes('/i') x(i)
						) s
						WHERE TRY_CAST(s.basket_id AS INT) = bi.BasketId
					)
				) THEN 'contract_signed'

				WHEN EXISTS (
					SELECT 1
					FROM [Tender].[logtender].[Tender] t
					WHERE EXISTS (
						SELECT 1
						FROM (
							SELECT LTRIM(RTRIM(x.i.value('.', 'VARCHAR(20)'))) AS basket_id
							FROM (
								SELECT CAST(
									'<i>' + REPLACE(t.Basket_Ids, ',', '</i><i>') + '</i>'
									AS XML
								) AS xmlData
							) a
							CROSS APPLY xmlData.nodes('/i') x(i)
						) s
						WHERE TRY_CAST(s.basket_id AS INT) = bi.BasketId
					)
				) THEN 'tender_started'

				WHEN bi.AddedAt IS NOT NULL THEN 'planned_by_specialist'
				WHEN bi.udate IS NOT NULL THEN 'nh_assigned'
				WHEN bi.cdate IS NOT NULL THEN 'department_to_nh'
				WHEN bi.dedate IS NOT NULL THEN 'sent_to_department'
				WHEN bi.ddate IS NOT NULL THEN 'created_by_company'

				ELSE 'created_by_company'
			END AS current_step_key

		FROM [Tender].[logtender].[BasketItems] bi
		WHERE ISNULL(bi.IsReturned, 0) = 0
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	items := []BasketItem{}

	for rows.Next() {
		var it BasketItem
		err := rows.Scan(
			&it.BasketItemId,
			&it.BasketId,
			&it.AddedAt,
			&it.Code,
			&it.Cr4Name,
			&it.CrMarkName,
			&it.DCode,
			&it.DName,
			&it.MDocNo,
			&it.MName,
			&it.USize,
			&it.Qty,
			&it.Price,
			&it.PriceSum,
			&it.PkgNo,
			&it.PkgName,
			&it.PkgDate,
			&it.DeathDate,
			&it.IsArrived,
			&it.Tailbar,
			&it.CurrentStepKey,
		)
		if err != nil {
			fmt.Println("❌ Scan error:", err)
			continue
		}
		items = append(items, it)
	}

	c.Data["json"] = items
	c.ServeJSON()
}
