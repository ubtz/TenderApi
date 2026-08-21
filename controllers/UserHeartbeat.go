package controllers

import (
	"net/http"
	"time"

	"github.com/astaxie/beego"
)

type UserHeartbeat struct {
	beego.Controller
}

func (c *UserHeartbeat) Post() {
	claims, err := getClaimsFromBearer(c.Ctx.Input.Header("Authorization"))
	if err != nil || claims.SessionID == "" {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]string{"error": "Invalid or missing token"}
		c.ServeJSON()
		return
	}

	tokenString, err := createAuthToken(claims.Email, claims.UserID, claims.SessionID, claims.Role)
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to refresh token"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]interface{}{
		"success":    true,
		"token":      tokenString,
		"expires_in": int(authTokenDuration / time.Second),
	}
	c.ServeJSON()
}
