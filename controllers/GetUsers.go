package controllers

import (
	"log"
	"net/http"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

// User struct matching your Users table
type Users struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	// PasswordHash string `json:"password_hash"`
	Ovog      string `json:"ovog"`
	Ner       string `json:"ner"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	Code      string `json:"code"`
	Dep       string `json:"dep"`
	Div       string `json:"division"`
	Erh       string `json:"erh"`
}

// GetUsers controller
type GetUsers struct {
	beego.Controller
}

// GET /get/users
func (c *GetUsers) GetAllUsers() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
		SELECT TOP (1000) [Id], [Username], [Ovog], [Ner], 
		[Email], [CreatedAt], [Code], [Dep],[division], [Erh] 
		FROM [Tender].[dbo].[Users]
	`

	if config.Env == "prod" {
		query = `
			SELECT TOP (1000) [Id], [Username], [Ovog], [Ner], 
			[Email], [CreatedAt], [Code], [Dep],[division], [Erh] 
			FROM [Tender].[logtender].[Users]
		`
	}

	rows, err := db.Query(query)
	if err != nil {
		log.Println("Query error:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Query failed"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	var users []Users
	for rows.Next() {
		var u Users
		err := rows.Scan(&u.Id, &u.Username, &u.Ovog, &u.Ner,
			&u.Email, &u.CreatedAt, &u.Code, &u.Dep, &u.Div, &u.Erh)
		if err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		log.Println("Row iteration error:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to read rows"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = users
	c.ServeJSON()
}
