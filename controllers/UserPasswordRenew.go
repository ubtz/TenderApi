package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"net/http"

	"github.com/astaxie/beego"
)

type UserPasswordRenew struct {
	beego.Controller
}

type PasswordRenewRequest struct {
	UserId int `json:"userId"`
}

func (c *UserPasswordRenew) Post() {
	var req PasswordRenewRequest

	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &req); err != nil {
		beego.Error("❌ JSON unmarshal error:", err)
		c.CustomAbort(http.StatusBadRequest, "Invalid request body")
		return
	}
	claims, claimsErr := ClaimsForController(&c.Controller)
	if claimsErr != nil || claims.Role != "Удирдлага" {
		c.CustomAbort(http.StatusForbidden, "Management role is required")
		return
	}

	if req.UserId == 0 {
		c.CustomAbort(http.StatusBadRequest, "Missing userId")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// ✅ Check if user exists
	var userCount int
	checkQuery := `SELECT COUNT(*) FROM [Tender].[dbo].[Users] WHERE Id = @p1`
	if config.Env == "prod" {
		checkQuery = `SELECT COUNT(*) FROM [Tender].[logtender].[Users] WHERE Id = @p1`
	}

	err := db.QueryRow(checkQuery, req.UserId).Scan(&userCount)
	if err != nil {
		beego.Error("❌ Database error:", err)
		c.CustomAbort(http.StatusInternalServerError, "Database error")
		return
	}
	if userCount == 0 {
		c.CustomAbort(http.StatusNotFound, "User not found")
		return
	}

	temporaryPassword, passwordErr := newSessionID()
	if passwordErr != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to generate temporary password")
		return
	}
	newPassword := temporaryPassword[:12]
	newHash, hashErr := hashPassword(newPassword)
	if hashErr != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to secure password")
		return
	}

	updateQuery := `UPDATE [Tender].[dbo].[Users] SET PasswordHash = @p1 WHERE Id = @p2`
	if config.Env == "prod" {
		updateQuery = `UPDATE [Tender].[logtender].[Users] SET PasswordHash = @p1 WHERE Id = @p2`
	}

	_, err = db.Exec(updateQuery, newHash, req.UserId)
	if err != nil {
		beego.Error("❌ Failed to reset password:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to reset password")
		return
	}

	c.Data["json"] = map[string]string{
		"message":            "Password reset successfully",
		"temporary_password": newPassword,
	}
	c.ServeJSON()
}
