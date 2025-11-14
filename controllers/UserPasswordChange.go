package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/astaxie/beego"
)

type UserPasswordChange struct {
	beego.Controller
}

type PasswordChangeRequest struct {
	UserId      int    `json:"userId"`
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

func (c *UserPasswordChange) Post() {
	var req PasswordChangeRequest

	// 🔍 Log request body for debugging
	beego.Info("📩 Raw body:", string(c.Ctx.Input.RequestBody))

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		beego.Error("❌ JSON unmarshal error:", err)
		c.CustomAbort(http.StatusBadRequest, "Invalid request body")
		return
	}

	beego.Info("🧠 Parsed request:", req)

	if req.UserId == 0 || req.OldPassword == "" || req.NewPassword == "" {
		c.CustomAbort(http.StatusBadRequest, "Missing required fields")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	var storedHash string
	query := `SELECT PasswordHash FROM [Tender].[dbo].[Users] WHERE Id = @p1`
	if config.Env == "prod" {
		query = `SELECT PasswordHash FROM [Tender].[logtender].[Users] WHERE Id = @p1`
	}

	err := db.QueryRow(query, req.UserId).Scan(&storedHash)
	if err == sql.ErrNoRows {
		c.CustomAbort(http.StatusNotFound, "User not found")
		return
	} else if err != nil {
		beego.Error("❌ Database error:", err)
		c.CustomAbort(http.StatusInternalServerError, "Database error")
		return
	}

	// Compare old password
	oldHash := hashPassword(req.OldPassword)
	if storedHash != oldHash {
		c.CustomAbort(http.StatusBadRequest, "Old password is incorrect")
		return
	}

	// Hash and update new password
	newHash := hashPassword(req.NewPassword)
	updateQuery := `UPDATE [Tender].[dbo].[Users] SET PasswordHash = @p1 WHERE Id = @p2`
	if config.Env == "prod" {
		updateQuery = `UPDATE [Tender].[logtender].[Users] SET PasswordHash = @p1 WHERE Id = @p2`
	}

	_, err = db.Exec(updateQuery, newHash, req.UserId)
	if err != nil {
		beego.Error("❌ Failed to update password:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to update password")
		return
	}

	beego.Info("✅ Password changed successfully for user:", req.UserId)

	c.Data["json"] = map[string]string{"message": "Password updated successfully"}
	c.ServeJSON()
}
