package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/astaxie/beego"
)

type GetUserinfoById struct {
	beego.Controller
}

func (c *GetUserinfoById) Post() {
	var req struct {
		UserId int `json:"userId"`
	}

	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		c.CustomAbort(http.StatusBadRequest, "Empty request body")
		return
	}

	if err := json.Unmarshal(body, &req); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid request format")
		return
	}

	if req.UserId == 0 {
		c.CustomAbort(http.StatusBadRequest, "Missing userId")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	var user struct {
		Id        int
		Username  string
		Ovog      sql.NullString
		Ner       sql.NullString
		Email     sql.NullString
		CreatedAt sql.NullTime
		Code      sql.NullInt64
		Dep       sql.NullString
		Erh       sql.NullString
	}

	query := `
		SELECT Id, Username, Ovog, Ner, Email, CreatedAt, Code, Dep, Erh
		FROM [Tender].[dbo].[Users]
		WHERE Id = @p1
	`
	if config.Env == "prod" {
		query = `
			SELECT Id, Username, Ovog, Ner, Email, CreatedAt, Code, Dep, Erh
			FROM [Tender].[logtender].[Users]
			WHERE Id = @p1
		`
	}

	err := db.QueryRow(query, req.UserId).Scan(
		&user.Id,
		&user.Username,
		&user.Ovog,
		&user.Ner,
		&user.Email,
		&user.CreatedAt,
		&user.Code,
		&user.Dep,
		&user.Erh,
	)
	if err == sql.ErrNoRows {
		c.CustomAbort(http.StatusNotFound, "User not found")
		return
	} else if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	c.Data["json"] = map[string]interface{}{
		"id":         user.Id,
		"username":   user.Username,
		"last_name":  user.Ovog.String,
		"first_name": user.Ner.String,
		"email":      user.Email.String,
		"created_at": user.CreatedAt.Time,
		"code":       user.Code.Int64,
		"dep":        user.Dep.String,
		"erh":        user.Erh.String,
	}

	c.ServeJSON()
}
