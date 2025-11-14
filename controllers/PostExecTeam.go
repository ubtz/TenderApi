package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type PostExecTeam struct {
	beego.Controller
}

// ✅ INPUT STRUCT — фронт-оос ирэх payload
type ExecTeamInput struct {
	PlanRootNumber string `json:"planRootNumber"`
	Batlah         string `json:"batlah"`
	Zuvshuursun    string `json:"zuvshuursun"`
	Guitsetgesen   string `json:"guitsetgesen"`
	UserId         int    `json:"userId"` // ✅ Added
}

// ✅ POST /execTeam
func (c *PostExecTeam) PostExecTeam() {
	fmt.Println("📥 PostExecTeam endpoint hit")

	body := c.Ctx.Input.RequestBody
	fmt.Println("📩 Raw Body:", string(body))

	if len(body) == 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	var input ExecTeamInput
	if err := json.Unmarshal(body, &input); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON"}
		c.ServeJSON()
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// --- 1. Delete old row (only for that user)
	deleteQuery := `
	DELETE FROM [Tender].[dbo].[ExecTeam]
	WHERE PlanRootNumber = @p1 AND UserId = @p2
`
	if config.Env == "prod" {
		deleteQuery = `
		DELETE FROM [Tender].[logtender].[ExecTeam]
		WHERE PlanRootNumber = @p1 AND UserId = @p2
	`
	}

	_, err := db.Exec(deleteQuery, input.PlanRootNumber, input.UserId)
	if err != nil {
		fmt.Println("❌ Failed to delete old ExecTeam:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to delete old ExecTeam"}
		c.ServeJSON()
		return
	}

	// --- 2. Insert new row
	insertQuery := `
		INSERT INTO [Tender].[dbo].[ExecTeam] (
			PlanRootNumber, Батлах, Зөвшөөрсөн, Гүйцэтгэсэн, UserId
		)
		VALUES (@p1, @p2, @p3, @p4, @p5)
	`
	if config.Env == "prod" {
		insertQuery = `
			INSERT INTO [Tender].[logtender].[ExecTeam] (
				PlanRootNumber, Батлах, Зөвшөөрсөн, Гүйцэтгэсэн, UserId
			)
			VALUES (@p1, @p2, @p3, @p4, @p5)
		`
	}

	_, err = db.Exec(insertQuery,
		input.PlanRootNumber,
		input.Batlah,
		input.Zuvshuursun,
		input.Guitsetgesen,
		input.UserId, // ✅ Added
	)
	if err != nil {
		fmt.Println("❌ Failed to insert ExecTeam:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to insert ExecTeam"}
		c.ServeJSON()
		return
	}

	fmt.Println("✅ ExecTeam inserted successfully:", input.PlanRootNumber)
	c.Data["json"] = map[string]string{"message": "ExecTeam created successfully"}
	c.ServeJSON()
}
