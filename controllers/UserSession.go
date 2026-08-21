package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	mssql "github.com/denisenkom/go-mssqldb"
	"github.com/golang-jwt/jwt/v4"
)

type UserLogout struct {
	beego.Controller
}

func openDB(cfg DBConfig) (*sql.DB, error) {
	connString := fmt.Sprintf(
		"server=%s;port=%d;user id=%s;password=%s;database=%s;encrypt=disable",
		cfg.Server, cfg.Port, cfg.User, cfg.Password, cfg.Database,
	)

	connector, err := mssql.NewConnector(connString)
	if err != nil {
		return nil, err
	}

	db := sql.OpenDB(connector)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func sessionTable() string {
	return getSchema() + ".[UserSessionLog]"
}

func createUserSession(db *sql.DB, userID int, username, email, sessionID string) error {
	_, err := db.Exec(`
		INSERT INTO `+sessionTable()+` (
			UserId, Username, Email, SessionId,
			LoginTime, LastSeenTime, Status
		)
		VALUES (@p1, @p2, @p3, @p4, GETDATE(), GETDATE(), @p5)
	`, userID, username, email, sessionID, "active")

	return err
}

func touchUserSession(sessionID string) error {
	if sessionID == "" {
		return nil
	}

	db, err := openDB(getConfig(config.Env))
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		UPDATE `+sessionTable()+`
		SET LastSeenTime = GETDATE()
		WHERE SessionId = @p1 AND Status = @p2
	`, sessionID, "active")

	return err
}

func closeUserSession(sessionID string) error {
	if sessionID == "" {
		return nil
	}

	db, err := openDB(getConfig(config.Env))
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		UPDATE `+sessionTable()+`
		SET
			LastSeenTime = GETDATE(),
			LogoutTime = GETDATE(),
			DurationSeconds = DATEDIFF(SECOND, LoginTime, GETDATE()),
			Status = @p2
		WHERE SessionId = @p1 AND Status = @p3
	`, sessionID, "logout", "active")

	return err
}

func getClaimsFromBearer(authHeader string) (*Claims, error) {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return nil, fmt.Errorf("missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, fmt.Errorf("invalid authorization header")
	}

	claims := &Claims{}
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}
	token, err := jwt.ParseWithClaims(parts[1], claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}

func RequireAuth(ctx *context.Context) {
	if ctx.Input.Method() == http.MethodOptions ||
		ctx.Input.URL() == "/post/login" ||
		ctx.Input.URL() == "/get/CposCodeMeasurements" {
		return
	}

	claims, err := getClaimsFromBearer(ctx.Input.Header("Authorization"))
	if err != nil || claims.UserID == 0 {
		ctx.Output.SetStatus(http.StatusUnauthorized)
		ctx.Output.JSON(map[string]string{"error": "Invalid or missing token"}, false, false)
		return
	}

	ctx.Input.SetData("authClaims", claims)

	managementOnly := map[string]bool{
		"/post/register":          true,
		"/post/UserPasswordRenew": true,
		"/put/UserInfoUpdate":     true,
		"/post/upload":            true,
		"/delete/file":            true,
		"/external/employee":      true,
	}
	if managementOnly[ctx.Input.URL()] && claims.Role != "Удирдлага" {
		ctx.Output.SetStatus(http.StatusForbidden)
		ctx.Output.JSON(map[string]string{"error": "Management role is required"}, false, false)
	}
}

func ClaimsForController(c *beego.Controller) (*Claims, error) {
	if claims, ok := c.Ctx.Input.GetData("authClaims").(*Claims); ok && claims != nil {
		return claims, nil
	}
	return getClaimsFromBearer(c.Ctx.Input.Header("Authorization"))
}

func TrackSessionActivity(ctx *context.Context) {
	claims, err := getClaimsFromBearer(ctx.Input.Header("Authorization"))
	if err != nil || claims.SessionID == "" {
		return
	}

	go func(sessionID string) {
		if err := touchUserSession(sessionID); err != nil {
			log.Println("Failed to update user session activity:", err)
		}
	}(claims.SessionID)
}

func (c *UserLogout) Post() {
	claims, err := getClaimsFromBearer(c.Ctx.Input.Header("Authorization"))
	if err != nil || claims.SessionID == "" {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]string{"error": "Invalid or missing token"}
		c.ServeJSON()
		return
	}

	if err := closeUserSession(claims.SessionID); err != nil {
		log.Println("Failed to close user session:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to close session"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]interface{}{
		"success":    true,
		"message":    "Logged out successfully",
		"session_id": claims.SessionID,
	}
	c.ServeJSON()
}
