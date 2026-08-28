package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/astaxie/beego"
)

// =======================
// Controller
// =======================

type GetItemSteps struct {
	beego.Controller
}

// =======================
// Response Model
// =======================

type ItemStep struct {
	StepKey   string         `json:"step_key"`
	Title     string         `json:"title"`
	Date      sql.NullString `json:"date,omitempty"`
	IsCurrent bool           `json:"is_current"`
}

// =======================
// GET /get/GetItemSteps/:itemId
// =======================

func (c *GetItemSteps) GetItemSteps() {

	// --------------------------------------------------
	// 0️⃣ Validate itemId
	// --------------------------------------------------
	itemIdStr := c.Ctx.Input.Param(":itemId")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("📥 GetItemSteps START")
	fmt.Println("➡ itemId =", itemIdStr)

	itemId, err := strconv.Atoi(itemIdStr)
	if err != nil {
		c.CustomAbort(http.StatusBadRequest, "invalid itemId")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// --------------------------------------------------
	// 1️⃣ Get BasketItem dates
	// --------------------------------------------------
	itemQuery := `
	SELECT
		BasketId,
		ddate,
		dedate,
		cdate,
		udate,
		AddedAt,
		isArrived
	FROM [Tender].[dbo].[BasketItems]
	WHERE BasketItemId = @itemId AND ISNULL(IsReturned, 0) = 0
	`

	if config.Env == "prod" {
		itemQuery = `
		SELECT
			BasketId,
			ddate,
			dedate,
			cdate,
			udate,
			AddedAt,
			isArrived
		FROM [Tender].[logtender].[BasketItems]
		WHERE BasketItemId = @itemId AND ISNULL(IsReturned, 0) = 0
		`
	}

	var (
		basketId  int
		ddate     sql.NullString
		dedate    sql.NullString
		cdate     sql.NullString
		udate     sql.NullString
		addedAt   sql.NullString
		isArrived sql.NullBool
	)

	err = db.QueryRow(itemQuery, sql.Named("itemId", itemId)).Scan(
		&basketId,
		&ddate,
		&dedate,
		&cdate,
		&udate,
		&addedAt,
		&isArrived,
	)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}

	// --------------------------------------------------
	// 2️⃣ Build item-side steps
	// --------------------------------------------------
	steps := []ItemStep{}

	if ddate.Valid {
		steps = append(steps, ItemStep{
			StepKey: "created_by_company",
			Title:   "ААН үүсгэсэн",
			Date:    ddate,
		})
	}

	if dedate.Valid {
		steps = append(steps, ItemStep{
			StepKey: "sent_to_department",
			Title:   "ААН албаруу шилжүүлсэн",
			Date:    dedate,
		})
	}

	if cdate.Valid {
		steps = append(steps, ItemStep{
			StepKey: "department_to_nh",
			Title:   "Алба НХ албаруу шилжүүлсэн",
			Date:    cdate,
		})
	}

	if udate.Valid {
		steps = append(steps, ItemStep{
			StepKey: "nh_assigned",
			Title:   "НХ алба мэргэжилтнүүдэд хувиарласан",
			Date:    udate,
		})
	}

	if addedAt.Valid {
		steps = append(steps, ItemStep{
			StepKey: "planned_by_specialist",
			Title:   "Мэргэжилтэн төлөвлөгөөндөө оруулсан",
			Date:    addedAt,
		})
	}

	// Mark last item-step as current (temporary)
	for i := range steps {
		steps[i].IsCurrent = false
	}
	if len(steps) > 0 {
		steps[len(steps)-1].IsCurrent = true
	}

	// --------------------------------------------------
	// 3️⃣ Check Tender
	// --------------------------------------------------
	tenderQuery := `
	SELECT TOP 1
    t.TenderId,
    t.Тендер_нээх_огноо
		FROM [Tender].[dbo].[Tender] t
		WHERE ISNULL(t.IsDeleted, 0) = 0
		  AND EXISTS (
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
			WHERE TRY_CAST(s.basket_id AS INT) = @basketId
		)
		ORDER BY t.Тендер_нээх_огноо DESC

	`

	if config.Env == "prod" {
		tenderQuery = `
		SELECT TOP 1
    t.TenderId,
    t.Тендер_нээх_огноо
	FROM [Tender].[logtender].[Tender] t
	WHERE ISNULL(t.IsDeleted, 0) = 0
	  AND EXISTS (
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
		WHERE TRY_CAST(s.basket_id AS INT) = @basketId
	)
	ORDER BY t.Тендер_нээх_огноо DESC

		`
	}

	var (
		tenderId   sql.NullInt64
		tenderDate sql.NullString
	)

	_ = db.QueryRow(
		tenderQuery,
		sql.Named("basketId", basketId),
	).Scan(&tenderId, &tenderDate)

	// --------------------------------------------------
	// 4️⃣ Check Geree (FINAL)
	// --------------------------------------------------
	gereeQuery := `
	SELECT TOP 1
		g.GereeId,
		g.CreatedAt
	FROM [Tender].[dbo].[Geree] g
	WHERE EXISTS (
		SELECT 1
		FROM (
			SELECT LTRIM(RTRIM(x.i.value('.', 'VARCHAR(20)'))) AS basket_id
			FROM (
				SELECT CAST('<i>' + REPLACE(g.Basket_Ids, ',', '</i><i>') + '</i>' AS XML) AS xmlData
			) a
			CROSS APPLY xmlData.nodes('/i') x(i)
		) s
		WHERE TRY_CAST(s.basket_id AS INT) = @basketId
	)
	ORDER BY g.CreatedAt DESC
	`

	if config.Env == "prod" {
		gereeQuery = `
		SELECT TOP 1
			g.GereeId,
			g.CreatedAt
		FROM [Tender].[logtender].[Geree] g
		WHERE EXISTS (
			SELECT 1
			FROM (
				SELECT LTRIM(RTRIM(x.i.value('.', 'VARCHAR(20)'))) AS basket_id
				FROM (
					SELECT CAST('<i>' + REPLACE(g.Basket_Ids, ',', '</i><i>') + '</i>' AS XML) AS xmlData
				) a
				CROSS APPLY xmlData.nodes('/i') x(i)
			) s
			WHERE TRY_CAST(s.basket_id AS INT) = @basketId
		)
		ORDER BY g.CreatedAt DESC
		`
	}

	var (
		gereeId   sql.NullInt64
		gereeDate sql.NullString
	)

	_ = db.QueryRow(gereeQuery, sql.Named("basketId", basketId)).
		Scan(&gereeId, &gereeDate)

		// --------------------------------------------------
	// 5️⃣ FINAL workflow continuation
	// --------------------------------------------------

	// Reset current flags first
	for i := range steps {
		steps[i].IsCurrent = false
	}

	// 5.1️⃣ Tender comes AFTER "planned_by_specialist"
	if tenderId.Valid {
		steps = append(steps, ItemStep{
			StepKey: "tender_started",
			Title:   "Тендер зарласан",
			Date:    tenderDate,
		})
	}

	// 5.2️⃣ Geree comes AFTER tender
	if gereeId.Valid {
		steps = append(steps, ItemStep{
			StepKey: "contract_signed",
			Title:   "Гэрээ байгуулсан",
			Date:    gereeDate,
		})
	}

	// 5.3️⃣ Delivery is the final stage
	if isArrived.Valid && isArrived.Bool {
		steps = append(steps, ItemStep{
			StepKey: "delivered",
			Title:   "Бараа хүлээн авсан",
		})
	}

	// 5.4️⃣ Mark ONLY last step as current
	if len(steps) > 0 {
		steps[len(steps)-1].IsCurrent = true
	}

	// --------------------------------------------------
	// 6️⃣ Response
	// --------------------------------------------------
	fmt.Println("📤 Returning steps:", len(steps))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	c.Data["json"] = steps
	c.ServeJSON()
}
