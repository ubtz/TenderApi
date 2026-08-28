package controllers

import (
	"fmt"
	"log"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type Tender struct {
	TenderId                 int     `json:"tender_id"`
	PlanRootNumber           string  `json:"plan_root_number"`
	TenderName               string  `json:"tender_name"`
	ШалгаруулалтынТөрөл      string  `json:"шалгаруулалтын_төрөл"`
	ТендерийнДугаар          string  `json:"тендерийн_дугаар"`
	ТендерийнТөрөл           string  `json:"тендерийн_төрөл"`
	БатлагдсанТөсөвтӨртөг    float64 `json:"батлагдсан_төсөвт_өртөг"`
	УрилгийнДугаар           string  `json:"урилгийн_дугаар"`
	УрилгийнОгноо            string  `json:"урилгийн_огноо"`
	ҮнэлгээХийсэнОгноо       string  `json:"үнэлгээ_хийсэн_огноо"`
	МэдэгдэлТараасанОгноо    string  `json:"мэдэгдэл_тараасан_огноо"`
	ГэрээБайгуулахЭрхОлгосон string  `json:"гэрээ_байгуулах_эрх_олгосон_огноо"`
	ГомдолГаргасанОгноо      string  `json:"гомдол_гаргасан_огноо"`
	ТүтгэлзүүлсэнОгноо       string  `json:"түтгэлзүүлсэн_огноо"`
	ТендерАмжилттайБолсон    bool    `json:"тендер_амжилттай_болсон_эсэх"`
	ТендерийнЯвцШалтгаан     string  `json:"тендерийн_явц_шалтгаан"`
	Тайлбар                  string  `json:"тайлбар"`
	CreatedAt                string  `json:"created_at"`
	CreatedBy                int     `json:"created_by"`
	CreatedByOvog            string  `json:"Ovog"`
	CreatedByNer             string  `json:"Ner"`
	ТендерНээхОгноо          string  `json:"тендер_нээх_огноо"`
	ТендерHХаахОгноо         string  `json:"тендер_хаах_огноо"`
	ТендертОролцогч          string  `json:"тендерт_оролцогч"`
	Organization             string  `json:"Organization"`
	ҮДарга                   string  `json:"ү_дарга"`
	ҮГишүүд                  string  `json:"ү_гишүүд"`
	ҮДугаар                  string  `json:"ү_дугаар"`
	ҮОгноо                   string  `json:"ү_огноо"`
	ЗҮКДугаар                string  `json:"зүк_дугаар"`
	ЗҮКОгноо                 string  `json:"зүк_огноо"`
	BasketIds                string  `json:"basket_ids"`
	RootTenderId             *int    `json:"root_tender_id"`
}

type GetTender struct {
	beego.Controller
}

func (c *GetTender) GetTender() {
	fmt.Println("📥 GetTender endpoint hit")

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// 🧩 Build query depending on environment
	query := `
		SELECT TOP (100000)
			t.TenderId, t.PlanRootNumber, t.TenderName, t.[Шалгаруулалтын_төрөл],
			t.[Тендерийн_дугаар], t.[Тендерийн_төрөл], t.[Батлагдсан_төсөвт_өртөг],
			t.[Урилгийн_дугаар], t.[Урилгийн_огноо], t.[Үнэлгээ_хийсэн_огноо],
			t.[Мэдэгдэл_тараасан_огноо], t.[Гэрээ_байгуулах_эрх_олгосон_огноо],
			t.[Гомдол_гаргасан_огноо], t.[Түтгэлзүүлсэн_огноо],
			t.[Тендер_амжилттай_болсон_эсэх], t.[Тендерийн_явц_шалтгаан],
			t.[Тайлбар], t.CreatedAt, t.CreatedBy,
			ISNULL(u.Ovog, '') AS CreatedByOvog,
			ISNULL(u.Ner, '') AS CreatedByNer,
			t.[Тендер_нээх_огноо], t.[Тендер_хаах_огноо],
			t.[Тендерт_оролцогч], t.[Organization], t.[Ү_Дарга], t.[Ү_Гишүүд],
			t.[Ү_Дугаар], t.[Ү_Огноо], t.[ЗҮК_Дугаар], t.[ЗҮК_Огноо], t.[Basket_Ids], t.[RootTenderId]
		FROM [Tender].[dbo].[Tender] AS t
		LEFT JOIN [Tender].[dbo].[Users] AS u ON t.CreatedBy = u.Id
		WHERE ISNULL(t.IsDeleted, 0) = 0
		  AND ISNULL(t.IsCurrent, 1) = 1
		  AND ISNULL(t.LifecycleStatus, N'Active') = N'Active'
		ORDER BY t.TenderId DESC
	`

	if config.Env == "prod" {
		query = `
		SELECT TOP (100000)
			t.TenderId, t.PlanRootNumber, t.TenderName, t.[Шалгаруулалтын_төрөл],
			t.[Тендерийн_дугаар], t.[Тендерийн_төрөл], t.[Батлагдсан_төсөвт_өртөг],
			t.[Урилгийн_дугаар], t.[Урилгийн_огноо], t.[Үнэлгээ_хийсэн_огноо],
			t.[Мэдэгдэл_тараасан_огноо], t.[Гэрээ_байгуулах_эрх_олгосон_огноо],
			t.[Гомдол_гаргасан_огноо], t.[Түтгэлзүүлсэн_огноо],
			t.[Тендер_амжилттай_болсон_эсэх], t.[Тендерийн_явц_шалтгаан],
			t.[Тайлбар], t.CreatedAt, t.CreatedBy,
			ISNULL(u.Ovog, '') AS CreatedByOvog,
			ISNULL(u.Ner, '') AS CreatedByNer,
			t.[Тендер_нээх_огноо], t.[Тендер_хаах_огноо],
			t.[Тендерт_оролцогч], t.[Organization], t.[Ү_Дарга], t.[Ү_Гишүүд],
			t.[Ү_Дугаар], t.[Ү_Огноо], t.[ЗҮК_Дугаар], t.[ЗҮК_Огноо], t.[Basket_Ids], t.[RootTenderId]
		FROM [Tender].[logtender].[Tender] AS t
		LEFT JOIN [Tender].[logtender].[Users] AS u ON t.CreatedBy = u.Id
		WHERE ISNULL(t.IsDeleted, 0) = 0
		  AND ISNULL(t.IsCurrent, 1) = 1
		  AND ISNULL(t.LifecycleStatus, N'Active') = N'Active'
		ORDER BY t.TenderId DESC
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("❌ Failed to query tenders: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": err.Error()}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var tenders []Tender

	for rows.Next() {
		var t Tender
		err := rows.Scan(
			&t.TenderId,
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
			&t.RootTenderId,
		)
		if err != nil {
			log.Printf("❌ Row scan error: %v", err)
			continue
		}
		tenders = append(tenders, t)
	}

	c.Data["json"] = tenders
	c.ServeJSON()
}
