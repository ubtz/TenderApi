package controllers

import (
	config "TenderApi/conf"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/astaxie/beego"
	"github.com/golang-jwt/jwt/v4"
)

type UserLogin struct {
	beego.Controller
}

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

var jwtSecret = []byte("your_secret_key")

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func (c *UserLogin) PostLogin() {
	var loginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		log.Println("⚠️ Empty request body")
		c.Data["json"] = map[string]string{"error": "Empty request body"}
		c.ServeJSON()
		return
	}

	if err := json.Unmarshal(body, &loginRequest); err != nil {
		log.Println("❌ Invalid JSON format:", err)
		c.Data["json"] = map[string]string{"error": "Invalid request format"}
		c.ServeJSON()
		return
	}

	log.Printf("🔐 Login attempt: %s", loginRequest.Email)

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	var user struct {
		Id           int
		Username     string
		PasswordHash string
		Ovog         sql.NullString
		Ner          sql.NullString
		Email        sql.NullString
		CreatedAt    time.Time
		Code         int
		Erh          sql.NullString
	}

	// 🔍 Select correct DB depending on environment
	query := `SELECT Id, Username, PasswordHash, Ovog, Ner, Email, CreatedAt, Code, Erh
			  FROM [Tender].[dbo].[Users] WHERE Email = @p1`
	if config.Env == "prod" {
		query = `SELECT Id, Username, PasswordHash, Ovog, Ner, Email, CreatedAt, Code, Erh
			  FROM [Tender].[logtender].[Users] WHERE Email = @p1`
	}

	err := db.QueryRow(query, loginRequest.Email).Scan(
		&user.Id, &user.Username, &user.PasswordHash,
		&user.Ovog, &user.Ner, &user.Email, &user.CreatedAt, &user.Code, &user.Erh,
	)
	if err == sql.ErrNoRows {
		log.Println("⚠️ No user found for email:", loginRequest.Email)
		c.Data["json"] = map[string]string{"error": "User not found"}
		c.ServeJSON()
		return
	} else if err != nil {
		log.Println("❌ Database query error:", err)
		c.Data["json"] = map[string]string{"error": "Database error"}
		c.ServeJSON()
		return
	}

	if user.PasswordHash != hashPassword(loginRequest.Password) {
		log.Println("❌ Invalid password for:", loginRequest.Email)
		c.Data["json"] = map[string]string{"error": "Invalid credentials"}
		c.ServeJSON()
		return
	}

	// ✅ Generate JWT token
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: user.Email.String,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		log.Println("❌ Failed to generate JWT token:", err)
		c.CustomAbort(http.StatusInternalServerError, "Could not generate token")
		return
	}

	// ✅ Log successful login depending on environment
	var insertQuery string
	if config.Env == "prod" {
		insertQuery = `
			INSERT INTO [Tender].[logtender].[UserLoginLog] (LoginTime, UserId, Username, Ovog, Ner)
			VALUES (@p1, @p2, @p3, @p4, @p5)
		`
	} else {
		insertQuery = `
			INSERT INTO [Tender].[dbo].[UserLoginLog] (LoginTime, UserId, Username, Ovog, Ner)
			VALUES (@p1, @p2, @p3, @p4, @p5)
		`
	}

	_, err = db.Exec(insertQuery, time.Now(), user.Id, user.Username, user.Ovog.String, user.Ner.String)
	if err != nil {
		log.Println("⚠️ Failed to insert login log:", err)
	} else {
		log.Printf("📝 Login logged for user %s (%d)", user.Username, user.Id)
	}

	// ✅ Send response
	c.Data["json"] = map[string]interface{}{
		"token":   tokenString,
		"message": "Login successful",
		"user": map[string]interface{}{
			"id":         user.Id,
			"username":   user.Username,
			"last_name":  user.Ovog.String,
			"first_name": user.Ner.String,
			"email":      user.Email.String,
			"created_at": user.CreatedAt.Format(time.RFC3339),
			"code":       user.Code,
			"Erh":        user.Erh.String,
		},
	}
	c.ServeJSON()
}
