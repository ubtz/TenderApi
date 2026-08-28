package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type PostBasket struct {
	beego.Controller
}

type BasketInput struct {
	UserId         int         `json:"userId"`
	BasketName     string      `json:"basketName"`
	BasketNumber   interface{} `json:"basketNumber"`
	BasketType     string      `json:"basketType"`
	PlanName       string      `json:"planName"`
	PlanRootNumber int         `json:"planRootNumber"`
	PublishDate    string      `json:"publishDate"`
	SetDate        string      `json:"setDate"`
	IsTemp         bool        `json:"isTemp"`
}

func parseBasketDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{
		"2006-01-02",
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006/01/02",
	} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported basket date %q", value)
}

func (c *PostBasket) PostBasket() {
	start := time.Now()
	log.Println("📥 [PostBasket] request received")

	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		log.Println("❌ [PostBasket] empty request body")
		c.CustomAbort(http.StatusBadRequest, "Empty request body")
		return
	}
	var input BasketInput
	if err := json.Unmarshal(body, &input); err != nil {
		log.Println("❌ [PostBasket] invalid JSON:", err)
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON")
		return
	}
	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID == 0 {
		c.CustomAbort(http.StatusUnauthorized, "Invalid user session")
		return
	}
	input.UserId = claims.UserID

	log.Printf(
		"🧾 [PostBasket] payload userId=%d planRoot=%d type=%s name=%s",
		input.UserId,
		input.PlanRootNumber,
		input.BasketType,
		input.BasketName,
	)

	publishDate, err1 := parseBasketDate(input.PublishDate)
	setDate, err2 := parseBasketDate(input.SetDate)
	if err1 != nil || err2 != nil {
		log.Printf(
			"❌ [PostBasket] date parse error publish=%s set=%s",
			input.PublishDate,
			input.SetDate,
		)
		c.CustomAbort(http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD or ISO 8601")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to start basket transaction")
		return
	}
	defer tx.Rollback()

	lockResource := "TenderBasket:" + strconv.Itoa(input.UserId) + ":" + strconv.Itoa(input.PlanRootNumber)
	var lockResult int
	if err := tx.QueryRow(`
		DECLARE @result int;
		EXEC @result = sp_getapplock
			@Resource = @p1,
			@LockMode = 'Exclusive',
			@LockOwner = 'Transaction',
			@LockTimeout = 10000;
		SELECT @result;
	`, lockResource).Scan(&lockResult); err != nil || lockResult < 0 {
		c.CustomAbort(http.StatusConflict, "Could not lock basket creation")
		return
	}

	log.Println("🔌 [PostBasket] DB connected (env =", config.Env, ")")

	if input.IsTemp {
		tempQuery := `
			SELECT TOP (1) BasketId, CAST(BasketNumber AS INT)
			FROM [Tender].[dbo].[Basket]
			WHERE UserId = @p1 AND ISNULL(IsTemp, 0) = 1
			ORDER BY BasketId
		`
		if config.Env == "prod" {
			tempQuery = strings.Replace(tempQuery, "[Tender].[dbo]", "[Tender].[logtender]", 1)
		}

		var existingID int64
		var existingNumber int
		err := tx.QueryRow(tempQuery, input.UserId).Scan(&existingID, &existingNumber)
		if err == nil {
			c.Data["json"] = map[string]interface{}{
				"message":       "Temporary basket already exists",
				"basket_id":     existingID,
				"basket_number": existingNumber,
			}
			c.ServeJSON()
			return
		}
		if err != sql.ErrNoRows {
			log.Printf("Temporary basket lookup failed: %v", err)
			c.CustomAbort(http.StatusInternalServerError, "Failed to check temporary basket")
			return
		}
	}

	if !input.IsTemp {
		duplicateQuery := `
			SELECT COUNT(1)
			FROM [Tender].[dbo].[Basket]
			WHERE UserId = @p1
				AND PlanRootNumber = @p2
				AND ISNULL(IsTemp, 0) = 0
				AND LOWER(LTRIM(RTRIM(BasketName))) = LOWER(LTRIM(RTRIM(@p3)))
		`
		if config.Env == "prod" {
			duplicateQuery = strings.Replace(duplicateQuery, "[Tender].[dbo]", "[Tender].[logtender]", 1)
		}

		var duplicateCount int
		if err := tx.QueryRow(duplicateQuery, input.UserId, input.PlanRootNumber, input.BasketName).Scan(&duplicateCount); err != nil {
			log.Printf("Basket duplicate-name lookup failed: %v", err)
			c.CustomAbort(http.StatusInternalServerError, "Failed to validate basket name")
			return
		}
		if duplicateCount > 0 {
			c.CustomAbort(http.StatusConflict, "Ижил нэртэй багц энэ төлөвлөгөөнд аль хэдийн байна")
			return
		}
	}

	var nextNum int
	numQuery := `
		SELECT ISNULL(MAX(CAST(BasketNumber AS INT)), 0) + 1
		FROM [Tender].[dbo].[Basket]
		WHERE UserId = @p1 AND PlanRootNumber = @p2 AND BasketType = @p3
	`
	if config.Env == "prod" {
		numQuery = strings.Replace(numQuery, "[Tender].[dbo]", "[Tender].[logtender]", 1)
	}

	err = tx.QueryRow(
		numQuery,
		input.UserId,
		input.PlanRootNumber,
		input.BasketType,
	).Scan(&nextNum)

	if err != nil {
		log.Printf(
			"⚠️ [PostBasket] failed to calc BasketNumber (user=%d plan=%d type=%s): %v",
			input.UserId,
			input.PlanRootNumber,
			input.BasketType,
			err,
		)
		nextNum = 1
	}

	log.Printf("🧮 [PostBasket] next BasketNumber = %d", nextNum)

	insertQuery := `
		INSERT INTO [Tender].[dbo].[Basket] (
			UserId, BasketName, BasketNumber, BasketType,
			PlanName, PlanRootNumber,
			PublishDate, SetDate, AddedAt, isValid, isTemp
		)
		OUTPUT INSERTED.BasketId
		VALUES (
			@p1, @p2, @p3, @p4,
			@p5, @p6, @p7,
			@p8, GETDATE(), CAST(0 AS BIT), @p9
		)
	`
	if config.Env == "prod" {
		insertQuery = strings.Replace(insertQuery, "[Tender].[dbo]", "[Tender].[logtender]", 1)
	}

	var newID int64
	err = tx.QueryRow(
		insertQuery,
		input.UserId,
		input.BasketName,
		strconv.Itoa(nextNum),
		input.BasketType,
		input.PlanName,
		input.PlanRootNumber,
		publishDate,
		setDate,
		input.IsTemp,
	).Scan(&newID)

	if err != nil {
		log.Printf(
			"❌ [PostBasket] insert failed user=%d plan=%d type=%s err=%v",
			input.UserId,
			input.PlanRootNumber,
			input.BasketType,
			err,
		)
		c.CustomAbort(http.StatusInternalServerError, "Failed to insert basket")
		return
	}
	if err := tx.Commit(); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to commit basket")
		return
	}

	log.Printf(
		"✅ [PostBasket] success id=%d user=%d plan=%d type=%s num=%d (%s)",
		newID,
		input.UserId,
		input.PlanRootNumber,
		input.BasketType,
		nextNum,
		time.Since(start),
	)

	c.Ctx.Output.SetStatus(http.StatusCreated)
	c.Data["json"] = map[string]interface{}{
		"message":       "Basket created successfully",
		"basket_id":     newID,
		"basket_number": nextNum,
	}
	c.ServeJSON()
}
