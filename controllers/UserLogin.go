package controllers

import (
	config "TenderApi/conf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"github.com/golang-jwt/jwt/v4"
)

type UserLogin struct {
	beego.Controller
}

type Claims struct {
	Email     string `json:"email"`
	UserID    int    `json:"user_id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

const authTokenDuration = 15 * time.Minute

var (
	developmentJWTSecret     []byte
	developmentJWTSecretOnce sync.Once
)

func getJWTSecret() ([]byte, error) {
	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if len(secret) >= 32 {
		return []byte(secret), nil
	}
	if secret != "" || config.Env == "prod" {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}

	developmentJWTSecretOnce.Do(func() {
		developmentJWTSecret = make([]byte, 32)
		if _, err := rand.Read(developmentJWTSecret); err != nil {
			developmentJWTSecret = nil
			return
		}
		log.Println("WARNING: JWT_SECRET is not configured; using a temporary development secret")
	})
	if len(developmentJWTSecret) != 32 {
		return nil, fmt.Errorf("could not generate development JWT secret")
	}
	return developmentJWTSecret, nil
}

func legacyPasswordHash(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

const passwordHashIterations = 210000

func derivePasswordKey(password, salt []byte, iterations int) []byte {
	block := make([]byte, len(salt)+4)
	copy(block, salt)
	binary.BigEndian.PutUint32(block[len(salt):], 1)
	mac := hmac.New(sha256.New, password)
	mac.Write(block)
	current := mac.Sum(nil)
	derived := append([]byte(nil), current...)
	for iteration := 1; iteration < iterations; iteration++ {
		mac.Reset()
		mac.Write(current)
		current = mac.Sum(nil)
		for index := range derived {
			derived[index] ^= current[index]
		}
	}
	return derived
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := derivePasswordKey([]byte(password), salt, passwordHashIterations)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", passwordHashIterations, hex.EncodeToString(salt), hex.EncodeToString(hash)), nil
}

func verifyPassword(storedHash, password string) bool {
	parts := strings.Split(storedHash, "$")
	if len(parts) == 4 && parts[0] == "pbkdf2-sha256" {
		iterations, err := strconv.Atoi(parts[1])
		salt, saltErr := hex.DecodeString(parts[2])
		expected, hashErr := hex.DecodeString(parts[3])
		if err != nil || saltErr != nil || hashErr != nil || iterations < 100000 {
			return false
		}
		actual := derivePasswordKey([]byte(password), salt, iterations)
		return len(actual) == len(expected) && subtle.ConstantTimeCompare(actual, expected) == 1
	}
	actualLegacy := legacyPasswordHash(password)
	return len(storedHash) == len(actualLegacy) && subtle.ConstantTimeCompare([]byte(storedHash), []byte(actualLegacy)) == 1
}

func newSessionID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func createAuthToken(email string, userID int, sessionID, role string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}
	expirationTime := time.Now().Add(authTokenDuration)
	claims := &Claims{
		Email:     email,
		UserID:    userID,
		SessionID: sessionID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
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

	if !verifyPassword(user.PasswordHash, loginRequest.Password) {
		log.Println("❌ Invalid password for:", loginRequest.Email)
		c.Data["json"] = map[string]string{"error": "Invalid credentials"}
		c.ServeJSON()
		return
	}
	if !strings.HasPrefix(user.PasswordHash, "pbkdf2-sha256$") {
		if upgradedHash, hashErr := hashPassword(loginRequest.Password); hashErr == nil {
			updateQuery := `UPDATE [Tender].[dbo].[Users] SET PasswordHash = @p1 WHERE Id = @p2`
			if config.Env == "prod" {
				updateQuery = `UPDATE [Tender].[logtender].[Users] SET PasswordHash = @p1 WHERE Id = @p2`
			}
			if _, updateErr := db.Exec(updateQuery, upgradedHash, user.Id); updateErr != nil {
				log.Println("Failed to upgrade password hash:", updateErr)
			}
		}
	}

	// ✅ Generate JWT token
	sessionID, err := newSessionID()
	if err != nil {
		log.Println("Failed to generate session id:", err)
		c.CustomAbort(http.StatusInternalServerError, "Could not generate session")
		return
	}

	tokenString, err := createAuthToken(user.Email.String, user.Id, sessionID, user.Erh.String)
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

	if err := createUserSession(db, user.Id, user.Username, user.Email.String, sessionID); err != nil {
		log.Println("Failed to insert user session:", err)
	} else {
		log.Printf("Session started for user %s (%d), session=%s", user.Username, user.Id, sessionID)
	}

	// ✅ Send response
	c.Data["json"] = map[string]interface{}{
		"token":      tokenString,
		"session_id": sessionID,
		"message":    "Login successful",
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
