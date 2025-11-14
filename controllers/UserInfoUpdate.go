package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"net/http"

	"github.com/astaxie/beego"
)

type UserInfoUpdate struct {
	beego.Controller
}

// 🔹 Expected request JSON structure
type UserInfoUpdateRequest struct {
	Id    int    `json:"id"`
	Ovog  string `json:"ovog"`
	Ner   string `json:"ner"`
	Email string `json:"email"`
	Code  string `json:"code"`
	Dep   string `json:"dep"`
	Erh   string `json:"erh"`
}

func (c *UserInfoUpdate) Put() {
	var req UserInfoUpdateRequest

	// 🔍 Debug log request body
	beego.Info("📩 Raw body:", string(c.Ctx.Input.RequestBody))

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		beego.Error("❌ JSON unmarshal error:", err)
		c.CustomAbort(http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Id == 0 {
		c.CustomAbort(http.StatusBadRequest, "Missing user ID")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// ✅ Verify user exists
	var count int
	checkQuery := `SELECT COUNT(*) FROM [Tender].[dbo].[Users] WHERE Id = @p1`
	if config.Env == "prod" {
		checkQuery = `SELECT COUNT(*) FROM [Tender].[logtender].[Users] WHERE Id = @p1`
	}

	err := db.QueryRow(checkQuery, req.Id).Scan(&count)
	if err != nil {
		beego.Error("❌ Database error:", err)
		c.CustomAbort(http.StatusInternalServerError, "Database error")
		return
	}
	if count == 0 {
		c.CustomAbort(http.StatusNotFound, "User not found")
		return
	}

	// ✅ Update user information
	updateQuery := `
		UPDATE [Tender].[dbo].[Users]
		SET Ovog = @p1, Ner = @p2, Email = @p3, Code = @p4, Dep = @p5, Erh = @p6
		WHERE Id = @p7
	`
	if config.Env == "prod" {
		updateQuery = `
			UPDATE [Tender].[logtender].[Users]
			SET Ovog = @p1, Ner = @p2, Email = @p3, Code = @p4, Dep = @p5, Erh = @p6
			WHERE Id = @p7
		`
	}

	_, err = db.Exec(updateQuery, req.Ovog, req.Ner, req.Email, req.Code, req.Dep, req.Erh, req.Id)
	if err != nil {
		beego.Error("❌ Failed to update user info:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to update user info")
		return
	}

	beego.Info("✅ User info updated successfully:", req.Id)

	c.Data["json"] = map[string]string{
		"message": "User info updated successfully",
	}
	c.ServeJSON()
}
