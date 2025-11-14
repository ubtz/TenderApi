package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"fmt"

	"github.com/astaxie/beego"
)

type Register struct {
	beego.Controller
}

type User struct {
	Username     string `json:"username"` // <-- Add this
	Ner          string `json:"Ner"`
	Ovog         string `json:"Ovog"`
	Email        string `json:"email"`
	Dep          string `json:"dep"`  // Department
	Code         string `json:"code"` // User code
	Erh          string `json:"Erh"`
	Password     string `json:"password"`     // input password
	PasswordHash string `json:"passwordHash"` // hashed password
	Regno        string `json:"regno"`
	Department   string `json:"department"`
	Division     string `json:"division"`
	Sector       string `json:"sector"`
}

// Dummy hash function — use bcrypt or a secure hashing library in real use.

func (c *Register) PostRegister() {
	fmt.Println("📥 PostRegister endpoint hit")

	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	var newUser User
	err := json.Unmarshal(body, &newUser)
	if err != nil {
		c.Ctx.Output.SetStatus(400)
		c.Data["json"] = map[string]string{"error": "Invalid request format"}
		c.ServeJSON()
		return
	}

	newUser.PasswordHash = hashPassword(newUser.Password)

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	insertQuery := `
		INSERT INTO [Tender].[dbo].[Users] (
			Username, PasswordHash, Ovog, Ner, Email, CreatedAt, Dep, Code,Erh,regno,department,division,sector
		) VALUES (
			@p1, @p2, @p3, @p4, @p5, GETDATE() , @p6, @p7,@p8,@p9,@p10,@p11,@p12
		)
	`
	if config.Env == "prod" {
		insertQuery = `
		INSERT INTO [Tender].[logtender].[Users] (
			Username, PasswordHash, Ovog, Ner, Email, CreatedAt, Dep, Code,Erh,regno,department,division,sector
		) VALUES (
			@p1, @p2, @p3, @p4, @p5, GETDATE() , @p6, @p7,@p8,@p9,@p10,@p11,@p12
		)
	`
	}
	_, err = db.Exec(
		insertQuery,
		newUser.Username,
		newUser.PasswordHash,
		newUser.Ovog,
		newUser.Ner,
		newUser.Email,
		newUser.Dep,  // <-- Add this
		newUser.Code, // <-- Add this
		newUser.Erh,
		newUser.Regno,
		newUser.Department,
		newUser.Division,
		newUser.Sector,
	)

	if err != nil {
		fmt.Println("❌ Failed to insert user:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Insert failed"}
		c.ServeJSON()
		return
	}

	fmt.Println("✅ User registered successfully:", newUser.Email)
	c.Data["json"] = map[string]string{"message": "User registered successfully"}
	c.ServeJSON()
}
