package controllers

import (
	"database/sql"
	"log"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type Geree struct {
	GereeId                int     `json:"GereeId"`
	TenderId               int     `json:"TenderId"`
	ТендерийнДугаар        string  `json:"тендерийн_дугаар"`
	TenderName             string  `json:"tender_name"`
	ТендерийнТөрөл         string  `json:"тендерийн_төрөл"`
	CreatedAt              string  `json:"CreatedAt"`
	ГэрээнийДугаар         string  `json:"гэрээний_дугаар"`
	ГэрээБайгуулсанОгноо   string  `json:"гэрээ_байгуулсан_огноо"`
	ГэрээБайгуулсанААН     string  `json:"гэрээ_байгуулсан_ААН"`
	Бэлтгэн_нийлүүлэгч_ААН string  `json:"бэлтгэн_нийлүүлэгч_ААН"`
	ААНРегистер            string  `json:"ААН_регистер"`
	ХүчинтэйХугацаа        string  `json:"хүчинтэй_хугацаа"`
	Валют                  string  `json:"валют"`
	ГэрээнийДүн            float64 `json:"гэрээний_дүн"`
	ТөлбөрийнНөхцөл        string  `json:"төлбөрийн_нөхцөл"`
	ТөлбөрийнОгноо         string  `json:"төлбөрийн_огноо"`
	ТөлбөрХийхХугацаа      string  `json:"төлбөр_хийх_хугацаа"`
	НийлүүлэхНөхцөл        string  `json:"нийлүүлэх_нөхцөл"`
	НийлүүлэхХугацаа       string  `json:"нийлүүлэх_хугацаа"`
	АлдангийнНөхцөл        string  `json:"алдангийн_нөхцөл"`
	ГэрээХэрэгжилтийнЯвц   string  `json:"гэрээ_хэрэгжилтийн_явц"`
	Тодруулга              string  `json:"тодруулга"`
	Дүгнэлт                string  `json:"дүгнэлт"`
	Санамж                 string  `json:"санамж"`
	ГэрээнийТөлөв          string  `json:"гэрээний_төлөв"`
	АКТогноо               string  `json:"акт_огноо"`
	BasketIds              string  `json:"basket_ids"`
	GereeUserId            int     `json:"GereeUserId"`
	CreatedByUser          string  `json:"created_by_user"` // from Tender.CreatedBy
	GereeUserName          string  `json:"geree_user_name"` // from Geree.GereeUserId
}

type GetGeree struct {
	beego.Controller
}

func (c *GetGeree) GetGeree() {
	log.Println("📥 GetGeree endpoint hit")

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
	SELECT TOP (1000)
		g.[GereeId],
		g.[TenderId],
		t.[Тендерийн_дугаар],
		t.[TenderName],
		t.[Тендерийн_төрөл],
		g.[CreatedAt],
		g.[Гэрээний_дугаар],
		g.[Гэрээ_байгуулсан_огноо],
		g.[Гэрээ_байгуулсан_ААН],
		g.[Бэлтгэн_нийлүүлэгч_ААН],
		g.[ААН_регистер],
		g.[Хүчинтэй_хугацаа],
		g.[Валют],
		g.[Гэрээний_дүн],
		g.[Төлбөрийн_нөхцөл],
		g.[Төлбөрийн_огноо],
		g.[Төлбөр_хийх_хугацаа],
		g.[Нийлүүлэх_нөхцөл],
		g.[Нийлүүлэх_хугацаа],
		g.[Алдангийн_нөхцөл],
		g.[Гэрээ_хэрэгжилтийн_явц],
		g.[Тодруулга],
		g.[Дүгнэлт],
		g.[Санамж],
		g.[Гэрээний_төлөв],
		g.[АКТ_огноо],
		g.[Basket_Ids],
		g.[GereeUserId],
		ISNULL(u1.[Ovog],'') + ' ' + ISNULL(u1.[Ner],'') AS CreatedByUser,
		ISNULL(u2.[Ovog],'') + ' ' + ISNULL(u2.[Ner],'') AS GereeUserName
	FROM [Tender].[dbo].[Geree] g
	JOIN [Tender].[dbo].[Tender] t 
		ON g.[TenderId] = t.[TenderId]
	LEFT JOIN [Tender].[dbo].[Users] u1
		ON t.[CreatedBy] = u1.[Id]
	LEFT JOIN [Tender].[dbo].[Users] u2
		ON g.[GereeUserId] = u2.[Id]
	WHERE t.[Түтгэлзүүлсэн_огноо] = '1900-01-01T00:00:00Z'
	ORDER BY g.[CreatedAt] DESC;
	`

	if config.Env == "prod" {
		query = `
		SELECT TOP (1000)
			g.[GereeId],
			g.[TenderId],
			t.[Тендерийн_дугаар],
			t.[TenderName],
			t.[Тендерийн_төрөл],
			g.[CreatedAt],
			g.[Гэрээний_дугаар],
			g.[Гэрээ_байгуулсан_огноо],
			g.[Гэрээ_байгуулсан_ААН],
			g.[Бэлтгэн_нийлүүлэгч_ААН],
			g.[ААН_регистер],
			g.[Хүчинтэй_хугацаа],
			g.[Валют],
			g.[Гэрээний_дүн],
			g.[Төлбөрийн_нөхцөл],
			g.[Төлбөрийн_огноо],
			g.[Төлбөр_хийх_хугацаа],
			g.[Нийлүүлэх_нөхцөл],
			g.[Нийлүүлэх_хугацаа],
			g.[Алдангийн_нөхцөл],
			g.[Гэрээ_хэрэгжилтийн_явц],
			g.[Тодруулга],
			g.[Дүгнэлт],
			g.[Санамж],
			g.[Гэрээний_төлөв],
			g.[АКТ_огноо],
			g.[Basket_Ids],
			g.[GereeUserId],
			ISNULL(u1.[Ovog],'') + ' ' + ISNULL(u1.[Ner],'') AS CreatedByUser,
			ISNULL(u2.[Ovog],'') + ' ' + ISNULL(u2.[Ner],'') AS GereeUserName
		FROM [Tender].[logtender].[Geree] g
		JOIN [Tender].[logtender].[Tender] t 
			ON g.[TenderId] = t.[TenderId]
		LEFT JOIN [Tender].[logtender].[Users] u1
			ON t.[CreatedBy] = u1.[Id]
		LEFT JOIN [Tender].[logtender].[Users] u2
			ON g.[GereeUserId] = u2.[Id]
		WHERE t.[Түтгэлзүүлсэн_огноо] = '1900-01-01T00:00:00Z'
		ORDER BY g.[CreatedAt] DESC;
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Printf("❌ Query error: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": err.Error()}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var results []Geree

	for rows.Next() {
		var (
			gId, tId                                      int
			тендерийнДугаар, tenderName, tenderType       sql.NullString
			createdAt                                     sql.NullString
			gdugaar, gdate, gaan, bn                      sql.NullString
			reg, huu, val, tn, tognoo                     sql.NullString
			thh, nn, nh, an, gh                           sql.NullString
			tdr, dugnelt, sanамж, gt, АКТогноо, BasketIds sql.NullString
			gdun                                          sql.NullFloat64
			GereeUserId                                   int
			CreatedByUser, GereeUserName                  sql.NullString
		)

		err := rows.Scan(
			&gId, &tId, &тендерийнДугаар, &tenderName, &tenderType, &createdAt,
			&gdugaar, &gdate, &gaan, &bn,
			&reg, &huu, &val, &gdun,
			&tn, &tognoo, &thh,
			&nn, &nh, &an,
			&gh, &tdr, &dugnelt,
			&sanамж, &gt, &АКТогноо,
			&BasketIds, &GereeUserId,
			&CreatedByUser, &GereeUserName,
		)

		if err != nil {
			log.Printf("❌ Row scan error: %v", err)
			continue
		}

		results = append(results, Geree{
			GereeId:                gId,
			TenderId:               tId,
			ТендерийнДугаар:        nullToStr(тендерийнДугаар),
			TenderName:             nullToStr(tenderName),
			ТендерийнТөрөл:         nullToStr(tenderType),
			CreatedAt:              nullToStr(createdAt),
			ГэрээнийДугаар:         nullToStr(gdugaar),
			ГэрээБайгуулсанОгноо:   nullToStr(gdate),
			ГэрээБайгуулсанААН:     nullToStr(gaan),
			Бэлтгэн_нийлүүлэгч_ААН: nullToStr(bn),
			ААНРегистер:            nullToStr(reg),
			ХүчинтэйХугацаа:        nullToStr(huu),
			Валют:                  nullToStr(val),
			ГэрээнийДүн:            nullToFloat(gdun),
			ТөлбөрийнНөхцөл:        nullToStr(tn),
			ТөлбөрийнОгноо:         nullToStr(tognoo),
			ТөлбөрХийхХугацаа:      nullToStr(thh),
			НийлүүлэхНөхцөл:        nullToStr(nn),
			НийлүүлэхХугацаа:       nullToStr(nh),
			АлдангийнНөхцөл:        nullToStr(an),
			ГэрээХэрэгжилтийнЯвц:   nullToStr(gh),
			Тодруулга:              nullToStr(tdr),
			Дүгнэлт:                nullToStr(dugnelt),
			Санамж:                 nullToStr(sanамж),
			ГэрээнийТөлөв:          nullToStr(gt),
			АКТогноо:               nullToStr(АКТогноо),
			BasketIds:              nullToStr(BasketIds),
			GereeUserId:            GereeUserId,
			CreatedByUser:          nullToStr(CreatedByUser),
			GereeUserName:          nullToStr(GereeUserName),
		})
	}

	c.Data["json"] = results
	c.ServeJSON()
}

func nullToStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullToFloat(nf sql.NullFloat64) float64 {
	if nf.Valid {
		return nf.Float64
	}
	return 0
}
