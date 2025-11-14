package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

// ✅ Struct for Geree input
type GereeInput struct {
	TenderId               int      `json:"TenderId"`
	ААН_регистер           *string  `json:"ААН_регистер"`
	Алдангийн_нөхцөл       *string  `json:"Алдангийн_нөхцөл"`
	Валют                  *string  `json:"Валют"`
	Гэрээ_байгуулсан_ААН   *string  `json:"Гэрээ_байгуулсан_ААН"`
	Гэрээ_байгуулсан_огноо *string  `json:"Гэрээ_байгуулсан_огноо"`
	Гэрээ_хэрэгжилтийн_явц *string  `json:"Гэрээ_хэрэгжилтийн_явц"`
	Гэрээний_дугаар        *string  `json:"Гэрээний_дугаар"`
	Гэрээний_дүн           *float64 `json:"Гэрээний_дүн"`
	Гэрээний_төлөв         *string  `json:"Гэрээний_төлөв"`
	Дүгнэлт                *string  `json:"Дүгнэлт"`
	Нийлүүлэх_нөхцөл       *string  `json:"Нийлүүлэх_нөхцөл"`
	Нийлүүлэх_хугацаа      *string  `json:"Нийлүүлэх_хугацаа"`
	Санамж                 *string  `json:"Санамж"`
	Тодруулга              *string  `json:"Тодруулга"`
	Төлбөр_хийх_хугацаа    *string  `json:"Төлбөр_хийх_хугацаа"`
	Төлбөрийн_нөхцөл       *string  `json:"Төлбөрийн_нөхцөл"`
	Төлбөрийн_огноо        *string  `json:"Төлбөрийн_огноо"`
	Хүчинтэй_хугацаа       *string  `json:"Хүчинтэй_хугацаа"`
	Бэлтгэн_нийлүүлэгч_ААН *string  `json:"Бэлтгэн_нийлүүлэгч_ААН"`
	BasketIds              string   `json:"BasketIds"` // ✅ match frontend JSON key
	GereeUserId            int      `json:"GereeUserId"`
}

type PostGeree struct {
	beego.Controller
}

func (c *PostGeree) PostGeree() {
	fmt.Println("📥 PostGeree endpoint hit")

	// 🧾 Parse request body
	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	log.Printf("📦 Request body: %s", string(body))

	// ✅ Unmarshal JSON
	var input GereeInput
	if err := json.Unmarshal(body, &input); err != nil {
		log.Printf("❌ JSON unmarshal error: %v", err)
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON format"}
		c.ServeJSON()
		return
	}

	// ✅ Validate required field
	if input.TenderId == 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "TenderId is required"}
		c.ServeJSON()
		return
	}

	// 🧠 Debug values
	log.Printf("📋 Parsed input: %+v", input)

	// 🧠 Check BasketIds
	if input.BasketIds == "" {
		log.Println("⚠️ BasketIds is empty — will still insert as empty string.")
	}

	// 🗄️ DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// ✅ Build dynamic field map
	fields := map[string]interface{}{
		"TenderId":               input.TenderId,
		"ААН_регистер":           input.ААН_регистер,
		"Алдангийн_нөхцөл":       input.Алдангийн_нөхцөл,
		"Валют":                  input.Валют,
		"Гэрээ_байгуулсан_ААН":   input.Гэрээ_байгуулсан_ААН,
		"Гэрээ_байгуулсан_огноо": input.Гэрээ_байгуулсан_огноо,
		"Гэрээ_хэрэгжилтийн_явц": input.Гэрээ_хэрэгжилтийн_явц,
		"Гэрээний_дугаар":        input.Гэрээний_дугаар,
		"Гэрээний_дүн":           input.Гэрээний_дүн,
		"Гэрээний_төлөв":         input.Гэрээний_төлөв,
		"Дүгнэлт":                input.Дүгнэлт,
		"Нийлүүлэх_нөхцөл":       input.Нийлүүлэх_нөхцөл,
		"Нийлүүлэх_хугацаа":      input.Нийлүүлэх_хугацаа,
		"Санамж":                 input.Санамж,
		"Тодруулга":              input.Тодруулга,
		"Төлбөр_хийх_хугацаа":    input.Төлбөр_хийх_хугацаа,
		"Төлбөрийн_нөхцөл":       input.Төлбөрийн_нөхцөл,
		"Төлбөрийн_огноо":        input.Төлбөрийн_огноо,
		"Хүчинтэй_хугацаа":       input.Хүчинтэй_хугацаа,
		"Бэлтгэн_нийлүүлэгч_ААН": input.Бэлтгэн_нийлүүлэгч_ААН,
		"Basket_Ids":             input.BasketIds, // ✅ final field for DB
		"GereeUserId":            input.GereeUserId,
	}

	// 🧱 Build INSERT query
	var cols []string
	var params []string
	var values []interface{}

	for col, val := range fields {
		// skip nil pointers
		switch v := val.(type) {
		case *string:
			if v != nil {
				cols = append(cols, fmt.Sprintf("[%s]", col))
				params = append(params, fmt.Sprintf("@p%d", len(params)+1))
				values = append(values, *v)
			}
		case *float64:
			if v != nil {
				cols = append(cols, fmt.Sprintf("[%s]", col))
				params = append(params, fmt.Sprintf("@p%d", len(params)+1))
				values = append(values, *v)
			}
		case string:
			cols = append(cols, fmt.Sprintf("[%s]", col))
			params = append(params, fmt.Sprintf("@p%d", len(params)+1))
			values = append(values, v)
		case int:
			cols = append(cols, fmt.Sprintf("[%s]", col))
			params = append(params, fmt.Sprintf("@p%d", len(params)+1))
			values = append(values, v)
		}
	}

	// Add CreatedAt
	cols = append(cols, "[CreatedAt]")
	params = append(params, "GETDATE()")

	query := fmt.Sprintf(`INSERT INTO [Tender].[dbo].[Geree] (%s) VALUES (%s)`,
		strings.Join(cols, ", "),
		strings.Join(params, ", "),
	)
	if config.Env == "prod" {
		query = fmt.Sprintf(`INSERT INTO [Tender].[logtender].[Geree] (%s) VALUES (%s)`,
			strings.Join(cols, ", "),
			strings.Join(params, ", "),
		)
	}
	log.Printf("🧱 Final Query: %s", query)
	log.Printf("📦 Values: %+v", values)

	// ✅ Execute
	_, err := db.Exec(query, values...)
	if err != nil {
		log.Printf("❌ Failed to insert Geree: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": err.Error()}
		c.ServeJSON()
		return
	}

	log.Printf("✅ Geree inserted successfully for TenderId: %d", input.TenderId)
	c.Data["json"] = map[string]string{"message": "Гэрээ үүсгэгдлээ"}
	c.ServeJSON()
}
