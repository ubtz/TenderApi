package controllers

import (
	"database/sql"
	"log"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type GetTenderChainById struct {
	beego.Controller
}

func (c *GetTenderChainById) GetTenderChainById() {
	// 🔹 tenderId (эх эсвэл аль ч зангилаанаас эхэлж болно)
	tenderId := c.Ctx.Input.Param(":tenderId")
	if tenderId == "" {
		c.CustomAbort(http.StatusBadRequest, "tenderId is required")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// 🔹 Recursive CTE – CHAIN (previous → next)
	query := `
	WITH TenderChain AS (
    -- 🔹 Эхлэлийн тендер
    SELECT
        TenderId,
        RootTenderId,
        PlanRootNumber,
        TenderName,
        [Шалгаруулалтын_төрөл],
        [Тендерийн_дугаар],
        [Тендерийн_төрөл],
        [Батлагдсан_төсөвт_өртөг],
        [Урилгийн_дугаар],
        [Урилгийн_огноо],
        [Үнэлгээ_хийсэн_огноо],
        [Мэдэгдэл_тараасан_огноо],
        [Гэрээ_байгуулах_эрх_олгосон_огноо],
        [Гомдол_гаргасан_огноо],
        [Түтгэлзүүлсэн_огноо],
        [Тендер_амжилттай_болсон_эсэх],
        [Тендерийн_явц_шалтгаан],
        [Тайлбар],
        CreatedAt,
        CreatedBy,
        [Тендер_нээх_огноо],
        [Тендер_хаах_огноо],
        [Тендерт_оролцогч],
        [Organization],
        [Ү_Дарга],
        [Ү_Гишүүд],
        [Ү_Дугаар],
        [Ү_Огноо],
        [ЗҮК_Дугаар],
        [ЗҮК_Огноо],
        [Basket_Ids]
    FROM [Tender].[dbo].[Tender]
    WHERE TenderId = @tenderId

    UNION ALL

    -- 🔹 Recursive (ЗӨВХӨН INNER JOIN)
    SELECT
        t.TenderId,
        t.RootTenderId,
        t.PlanRootNumber,
        t.TenderName,
        t.[Шалгаруулалтын_төрөл],
        t.[Тендерийн_дугаар],
        t.[Тендерийн_төрөл],
        t.[Батлагдсан_төсөвт_өртөг],
        t.[Урилгийн_дугаар],
        t.[Урилгийн_огноо],
        t.[Үнэлгээ_хийсэн_огноо],
        t.[Мэдэгдэл_тараасан_огноо],
        t.[Гэрээ_байгуулах_эрх_олгосон_огноо],
        t.[Гомдол_гаргасан_огноо],
        t.[Түтгэлзүүлсэн_огноо],
        t.[Тендер_амжилттай_болсон_эсэх],
        t.[Тендерийн_явц_шалтгаан],
        t.[Тайлбар],
        t.CreatedAt,
        t.CreatedBy,
        t.[Тендер_нээх_огноо],
        t.[Тендер_хаах_огноо],
        t.[Тендерт_оролцогч],
        t.[Organization],
        t.[Ү_Дарга],
        t.[Ү_Гишүүд],
        t.[Ү_Дугаар],
        t.[Ү_Огноо],
        t.[ЗҮК_Дугаар],
        t.[ЗҮК_Огноо],
        t.[Basket_Ids]
    FROM [Tender].[dbo].[Tender] t
    INNER JOIN TenderChain tc
        ON t.RootTenderId = tc.TenderId
)
SELECT
    tc.*,
    ISNULL(u.Ovog, '') AS CreatedByOvog,
    ISNULL(u.Ner, '') AS CreatedByNer
FROM TenderChain tc
LEFT JOIN [Tender].[dbo].[Users] u
    ON tc.CreatedBy = u.Id
ORDER BY tc.CreatedAt;

	`

	if config.Env == "prod" {
		query = `
	WITH TenderChain AS (
		-- 🔹 Anchor
		SELECT
			TenderId,
			RootTenderId,
			PlanRootNumber,
			TenderName,
			[Шалгаруулалтын_төрөл],
			[Тендерийн_дугаар],
			[Тендерийн_төрөл],
			[Батлагдсан_төсөвт_өртөг],
			[Урилгийн_дугаар],
			[Урилгийн_огноо],
			[Үнэлгээ_хийсэн_огноо],
			[Мэдэгдэл_тараасан_огноо],
			[Гэрээ_байгуулах_эрх_олгосон_огноо],
			[Гомдол_гаргасан_огноо],
			[Түтгэлзүүлсэн_огноо],
			[Тендер_амжилттай_болсон_эсэх],
			[Тендерийн_явц_шалтгаан],
			[Тайлбар],
			CreatedAt,
			CreatedBy,
			[Тендер_нээх_огноо],
			[Тендер_хаах_огноо],
			[Тендерт_оролцогч],
			[Organization],
			[Ү_Дарга],
			[Ү_Гишүүд],
			[Ү_Дугаар],
			[Ү_Огноо],
			[ЗҮК_Дугаар],
			[ЗҮК_Огноо],
			[Basket_Ids]
		FROM [Tender].[logtender].[Tender]
		WHERE TenderId = @tenderId

		UNION ALL

		-- 🔹 Recursive (INNER JOIN ONLY)
		SELECT
			t.TenderId,
			t.RootTenderId,
			t.PlanRootNumber,
			t.TenderName,
			t.[Шалгаруулалтын_төрөл],
			t.[Тендерийн_дугаар],
			t.[Тендерийн_төрөл],
			t.[Батлагдсан_төсөвт_өртөг],
			t.[Урилгийн_дугаар],
			t.[Урилгийн_огноо],
			t.[Үнэлгээ_хийсэн_огноо],
			t.[Мэдэгдэл_тараасан_огноо],
			t.[Гэрээ_байгуулах_эрх_олгосон_огноо],
			t.[Гомдол_гаргасан_огноо],
			t.[Түтгэлзүүлсэн_огноо],
			t.[Тендер_амжилттай_болсон_эсэх],
			t.[Тендерийн_явц_шалтгаан],
			t.[Тайлбар],
			t.CreatedAt,
			t.CreatedBy,
			t.[Тендер_нээх_огноо],
			t.[Тендер_хаах_огноо],
			t.[Тендерт_оролцогч],
			t.[Organization],
			t.[Ү_Дарга],
			t.[Ү_Гишүүд],
			t.[Ү_Дугаар],
			t.[Ү_Огноо],
			t.[ЗҮК_Дугаар],
			t.[ЗҮК_Огноо],
			t.[Basket_Ids]
		FROM [Tender].[logtender].[Tender] t
		INNER JOIN TenderChain tc
			ON t.RootTenderId = tc.TenderId
	)
	SELECT
		tc.*,
		ISNULL(u.Ovog, '') AS CreatedByOvog,
		ISNULL(u.Ner, '') AS CreatedByNer
	FROM TenderChain tc
	LEFT JOIN [Tender].[logtender].[Users] u
		ON tc.CreatedBy = u.Id
	ORDER BY tc.CreatedAt
	`
	}

	rows, err := db.Query(query, sql.Named("tenderId", tenderId))
	if err != nil {
		log.Println("❌ Query error:", err)
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	var tenders []Tender

	for rows.Next() {
		var t Tender
		err := rows.Scan(
			&t.TenderId,
			&t.RootTenderId,
			&t.PlanRootNumber,
			&t.TenderName,
			&t.ШалгаруулалтынТөрөл,
			&t.ТендерийнДугаар,
			&t.ТендерийнТөрөл,
			&t.БатлагдсанТөсөвтӨртөг,
			&t.УрилгийнДугаар,
			&t.УрилгийнОгноо,
			&t.ҮнэлгээХийсэнОгноо,
			&t.МэдэгдэлТараасанОгноо,
			&t.ГэрээБайгуулахЭрхОлгосон,
			&t.ГомдолГаргасанОгноо,
			&t.ТүтгэлзүүлсэнОгноо,
			&t.ТендерАмжилттайБолсон,
			&t.ТендерийнЯвцШалтгаан,
			&t.Тайлбар,
			&t.CreatedAt,
			&t.CreatedBy,
			&t.CreatedByOvog,
			&t.CreatedByNer,
			&t.ТендерНээхОгноо,
			&t.ТендерHХаахОгноо,
			&t.ТендертОролцогч,
			&t.Organization,
			&t.ҮДарга,
			&t.ҮГишүүд,
			&t.ҮДугаар,
			&t.ҮОгноо,
			&t.ЗҮКДугаар,
			&t.ЗҮКОгноо,
			&t.BasketIds,
		)
		if err != nil {
			log.Println("❌ Scan error:", err)
			continue
		}
		tenders = append(tenders, t)
	}

	c.Data["json"] = tenders
	c.ServeJSON()
}
