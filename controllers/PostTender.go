package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

// Struct for Tender input
type TenderInput struct {
	PlanRootNumber           string  `json:"plan_root_number"`
	TenderName               string  `json:"tender_name"`
	ШалгаруулалтынТөрөл      string  `json:"шалгаруулалтын_төрөл"`
	ТендерийнДугаар          string  `json:"тендерийн_дугаар"`
	ТендерийнТөрөл           string  `json:"тендерийн_төрөл"`
	БатлагдсанТөсөвтӨртөг    float64 `json:"батлагдсан_төсөвт_өртөг"`
	УрилгийнДугаар           string  `json:"урилгын_дугаар"`
	УрилгийнОгноо            string  `json:"урилгын_огноо"`
	ҮнэлгээХийсэнОгноо       string  `json:"үнэлгээ_хийсэн_огноо"`
	МэдэгдэлТараасанОгноо    string  `json:"мэдэгдэл_тараасан_огноо"`
	ГэрээБайгуулахЭрхОлгосон string  `json:"гэрээ_байгуулах_эрх_олгосон_огноо"`
	ГомдолГаргасанОгноо      string  `json:"гомдол_гаргасан_огноо"`
	ТүтгэлзүүлсэнОгноо       string  `json:"түтгэлзүүлсэн_огноо"`
	ТендерАмжилттайБолсон    bool    `json:"тендер_амжилттай_болсон_эсэх"`
	ТендерийнЯвцШалтгаан     string  `json:"тендерийн_явц_шалтгаан"`
	Тайлбар                  string  `json:"тайлбар"`
	CreatedBy                int     `json:"CreatedBy"`
	ТендерНээхОгноо          string  `json:"тендер_нээх_огноо"` // new field
	ТендерHХаахОгноо         string  `json:"тендер_хаах_огноо"` // new field
	ТендертОролцогч          string  `json:"тендерт_оролцогч"`  // new field
	Organization             string  `json:"Organization"`      // new field
	ҮДарга                   string  `json:"ү_дарга"`           // new field
	ҮГишүүд                  string  `json:"ү_гишүүд"`          // new field
	ҮДугаар                  string  `json:"ү_дугаар"`          // new field
	ҮОгноо                   string  `json:"ү_огноо"`           // new field
	ЗҮКДугаар                string  `json:"зүк_дугаар"`        // new field
	ЗҮКОгноо                 string  `json:"зүк_огноо"`         // new field
	BasketIds                string  `json:"basket_ids"`        // new field for basket IDs
}

type PostTender struct {
	beego.Controller
}

// POST /post/PostTender
func (c *PostTender) PostTender() {
	fmt.Println("📥 PostTender endpoint hit")

	// Read request body
	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	log.Printf("📦 Request body: %s", string(body))

	var input TenderInput
	if err := json.Unmarshal(body, &input); err != nil {
		log.Printf("❌ JSON unmarshal error: %v", err)
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON"}
		c.ServeJSON()
		return
	}
	claims, claimsErr := ClaimsForController(&c.Controller)
	if claimsErr != nil || claims.UserID != input.CreatedBy {
		c.CustomAbort(http.StatusForbidden, "CreatedBy does not match authenticated session")
		return
	}

	// DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Insert query
	insertQuery := `
		INSERT INTO [Tender].[dbo].[Tender] (
			PlanRootNumber, TenderName, [Шалгаруулалтын_төрөл],
			[Тендерийн_дугаар], [Тендерийн_төрөл], [Батлагдсан_төсөвт_өртөг],
			[Урилгийн_дугаар], [Урилгийн_огноо], [Үнэлгээ_хийсэн_огноо],
			[Мэдэгдэл_тараасан_огноо], [Гэрээ_байгуулах_эрх_олгосон_огноо],
			[Гомдол_гаргасан_огноо], [Түтгэлзүүлсэн_огноо],
			[Тендер_амжилттай_болсон_эсэх], [Тендерийн_явц_шалтгаан],
			[Тайлбар], CreatedAt, CreatedBy,[Тендер_нээх_огноо],[Тендер_хаах_огноо],[Тендерт_оролцогч],[Organization],[Ү_Дарга],[Ү_Гишүүд],[Ү_Дугаар],[Ү_Огноо],[ЗҮК_Дугаар],[ЗҮК_Огноо],[Basket_Ids]
		)
		VALUES (
			@p1, @p2, @p3,
			@p4, @p5, @p6,
			@p7, @p8, @p9,
			@p10, @p11,
			@p12, @p13,
			@p14, @p15,
			@p16, GETDATE(), @p17,@p18,@p19,@p20,@p21,@p22,@p23,@p24,@p25,@p26,@p27,@p28
		)
	`
	if config.Env == "prod" {
		insertQuery = `
		INSERT INTO [Tender].[logtender].[Tender] (
			PlanRootNumber, TenderName, [Шалгаруулалтын_төрөл],
			[Тендерийн_дугаар], [Тендерийн_төрөл], [Батлагдсан_төсөвт_өртөг],
			[Урилгийн_дугаар], [Урилгийн_огноо], [Үнэлгээ_хийсэн_огноо],
			[Мэдэгдэл_тараасан_огноо], [Гэрээ_байгуулах_эрх_олгосон_огноо],
			[Гомдол_гаргасан_огноо], [Түтгэлзүүлсэн_огноо],
			[Тендер_амжилттай_болсон_эсэх], [Тендерийн_явц_шалтгаан],
			[Тайлбар], CreatedAt, CreatedBy,[Тендер_нээх_огноо],[Тендер_хаах_огноо],[Тендерт_оролцогч],[Organization],[Ү_Дарга],[Ү_Гишүүд],[Ү_Дугаар],[Ү_Огноо],[ЗҮК_Дугаар],[ЗҮК_Огноо],[Basket_Ids]
		)
		VALUES (
			@p1, @p2, @p3,
			@p4, @p5, @p6,
			@p7, @p8, @p9,
			@p10, @p11,
			@p12, @p13,
			@p14, @p15,
			@p16, GETDATE(), @p17,@p18,@p19,@p20,@p21,@p22,@p23,@p24,@p25,@p26,@p27,@p28
		)
	`
	}
	_, err := db.Exec(insertQuery,
		input.PlanRootNumber,
		input.TenderName,
		input.ШалгаруулалтынТөрөл,
		input.ТендерийнДугаар,
		input.ТендерийнТөрөл,
		input.БатлагдсанТөсөвтӨртөг,
		input.УрилгийнДугаар,
		input.УрилгийнОгноо,
		input.ҮнэлгээХийсэнОгноо,
		input.МэдэгдэлТараасанОгноо,
		input.ГэрээБайгуулахЭрхОлгосон,
		input.ГомдолГаргасанОгноо,
		input.ТүтгэлзүүлсэнОгноо,
		input.ТендерАмжилттайБолсон,
		input.ТендерийнЯвцШалтгаан,
		input.Тайлбар,
		input.CreatedBy,
		input.ТендерНээхОгноо,
		input.ТендерHХаахОгноо,
		input.ТендертОролцогч,
		input.Organization,
		input.ҮДарга,
		input.ҮГишүүд,
		input.ҮДугаар,
		input.ҮОгноо,
		input.ЗҮКДугаар,
		input.ЗҮКОгноо,
		input.BasketIds,
	)
	if err != nil {
		log.Printf("❌ Failed to insert tender: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": err.Error()} // return real DB error
		c.ServeJSON()
		return
	}

	log.Printf("✅ Tender inserted successfully: %s", input.TenderName)
	createNotificationSafe(
		db,
		claims.UserID,
		"tender_created",
		"Тендер үүсгэгдлээ",
		input.TenderName,
		"Tender",
		0,
		"/Tender",
	)
	c.Data["json"] = map[string]string{"message": "Tender created successfully"}
	c.ServeJSON()
}
