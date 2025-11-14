package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type PostUnelgeeHoroo struct {
	beego.Controller
}

// INPUT STRUCT — фронт-оос ирэх payload
type UnelgeeHorooInput struct {
	TenderId int    `json:"tenderId"`
	Darga    string `json:"darga"`
	Gishuud  string `json:"gishuud"`
	Dugaar   string `json:"dugaar"`
}

// POST /unelgeeHoroo
func (c *PostUnelgeeHoroo) PostUnelgeeHoroo() {
	fmt.Println("📥 PostUnelgeeHoroo endpoint hit")

	body := c.Ctx.Input.RequestBody
	fmt.Println("📩 Raw Body:", string(body))

	if len(body) == 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	var input UnelgeeHorooInput
	if err := json.Unmarshal(body, &input); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON"}
		c.ServeJSON()
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// --- 1. (Optional) Delete old row(s) by TenderId
	deleteQuery := `
		DELETE FROM [Tender].[dbo].[UnelgeeHoroo]
		WHERE TenderId = @p1
	`
	if config.Env == "prod" {
		deleteQuery = `
		DELETE FROM [Tender].[logtender].[UnelgeeHoroo]
		WHERE TenderId = @p1
		`
	}

	_, err := db.Exec(deleteQuery, input.TenderId)
	if err != nil {
		fmt.Println("❌ Failed to delete old UnelgeeHoroo:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to delete old UnelgeeHoroo"}
		c.ServeJSON()
		return
	}

	// --- 2. Insert new row
	insertQuery := `
		INSERT INTO [Tender].[dbo].[UnelgeeHoroo] (
			TenderId, Darga, Gishuud, Dugaar, Created
		)
		VALUES (@p1, @p2, @p3, @p4, GETDATE())
	`
	if config.Env == "prod" {
		insertQuery = `
		INSERT INTO [Tender].[logtender].[UnelgeeHoroo] (
			TenderId, Darga, Gishuud, Dugaar, Created
		)
		VALUES (@p1, @p2, @p3, @p4, GETDATE())
		`
	}

	_, err = db.Exec(insertQuery,
		input.TenderId,
		input.Darga,
		input.Gishuud,
		input.Dugaar,
	)
	if err != nil {
		fmt.Println("❌ Failed to insert UnelgeeHoroo:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to insert UnelgeeHoroo"}
		c.ServeJSON()
		return
	}

	fmt.Println("✅ UnelgeeHoroo inserted successfully:", input.TenderId)
	c.Data["json"] = map[string]string{"message": "UnelgeeHoroo created successfully"}
	c.ServeJSON()
}
