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

type UpdateTenderInput struct {
	Field string      `json:"field"`
	Value interface{} `json:"value"`
}

type UpdateTender struct {
	beego.Controller
}

// PUT /put/UpdateTender/:id
func (c *UpdateTender) Put() {
	tenderID := c.Ctx.Input.Param(":id")
	if tenderID == "" {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "TenderId is required"}
		c.ServeJSON()
		return
	}

	var input UpdateTenderInput
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Invalid JSON"}
		c.ServeJSON()
		return
	}

	if input.Field == "" {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Field name is required"}
		c.ServeJSON()
		return
	}

	// DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	wasSuspended := 0
	tenderName := ""
	isSuspensionField := strings.EqualFold(strings.TrimSpace(input.Field), "түтгэлзүүлсэн_огноо")
	if isSuspensionField {
		_ = db.QueryRow(`
			SELECT
				CASE WHEN CONVERT(NVARCHAR(10), [Түтгэлзүүлсэн_огноо], 23) = '1900-01-01' THEN 0 ELSE 1 END,
				ISNULL(TenderName, N'Тендер')
			FROM `+getSchema()+`.[Tender]
			WHERE TenderId = @p1
		`, tenderID).Scan(&wasSuspended, &tenderName)
	}

	// SQL injection хамгаалалттай query
	query := fmt.Sprintf(`UPDATE [Tender].[dbo].[Tender] SET [%s] = @p1 WHERE TenderId = @p2`, input.Field)
	if config.Env == "prod" {
		query = fmt.Sprintf(`UPDATE [Tender].[logtender].[Tender] SET [%s] = @p1 WHERE TenderId = @p2`, input.Field)
	}
	_, err := db.Exec(query, input.Value, tenderID)
	if err != nil {
		log.Printf("❌ Failed to update tender: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to update tender"}
		c.ServeJSON()
		return
	}

	valueText := strings.TrimSpace(fmt.Sprint(input.Value))
	isNowSuspended := isSuspensionField && valueText != "" && !strings.HasPrefix(valueText, "1900-01-01")
	if isNowSuspended && wasSuspended == 0 {
		rows, notificationErr := db.Query(`
			SELECT DISTINCT GereeUserId
			FROM `+getSchema()+`.[Geree]
			WHERE TenderId = @p1 AND ISNULL(GereeUserId, 0) > 0
		`, tenderID)
		if notificationErr != nil {
			log.Printf("Failed to find contract specialists for suspended tender %s: %v", tenderID, notificationErr)
		} else {
			defer rows.Close()
			for rows.Next() {
				var userID int
				if scanErr := rows.Scan(&userID); scanErr == nil {
					createNotificationSafe(
						db,
						userID,
						"tender_suspended",
						"Тендер түдгэлзүүлэгдлээ",
						tenderName,
						"Tender",
						0,
						"/",
					)
				}
			}
		}
	}

	c.Data["json"] = map[string]string{"message": "Tender updated successfully"}
	c.ServeJSON()
}
