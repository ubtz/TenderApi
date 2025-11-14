package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

// Controller struct
type PostBasket struct {
	beego.Controller
}

// Input struct
type BasketInput struct {
	UserId         int         `json:"userId"`
	BasketName     string      `json:"basketName"`
	BasketNumber   interface{} `json:"basketNumber"` // optional, ignored — backend generates
	BasketType     string      `json:"basketType"`
	PlanName       string      `json:"planName"`
	PlanRootNumber int         `json:"planRootNumber"`
	PublishDate    string      `json:"publishDate"` // YYYY-MM-DD
	SetDate        string      `json:"setDate"`     // YYYY-MM-DD
}

// POST /post/addBasket
func (c *PostBasket) PostBasket() {
	fmt.Println("📥 PostBasket endpoint hit")

	body := c.Ctx.Input.RequestBody
	if len(body) == 0 {
		c.CustomAbort(http.StatusBadRequest, "Empty request body")
		return
	}

	var input BasketInput
	if err := json.Unmarshal(body, &input); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON")
		return
	}

	// 🕒 Parse dates
	publishDate, err1 := time.Parse("2006-01-02", input.PublishDate)
	setDate, err2 := time.Parse("2006-01-02", input.SetDate)
	if err1 != nil || err2 != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid date format. Use YYYY-MM-DD")
		return
	}

	// ✅ DB connect
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// 🧮 Determine next BasketNumber (per user, plan, type)
	var nextNum int
	numQuery := `
		SELECT ISNULL(MAX(CAST(BasketNumber AS INT)), 0) + 1
		FROM [Tender].[dbo].[Basket]
		WHERE UserId = @p1 AND PlanRootNumber = @p2 AND BasketType = @p3
	`
	if config.Env == "prod" {
		numQuery = strings.Replace(numQuery, "[Tender].[dbo]", "[Tender].[logtender]", 1)
	}

	err := db.QueryRow(numQuery, input.UserId, input.PlanRootNumber, input.BasketType).Scan(&nextNum)
	if err != nil {
		fmt.Println("⚠️ Failed to get next BasketNumber:", err)
		nextNum = 1
	}

	// 🧾 Prepare insert query
	insertQuery := `
		INSERT INTO [Tender].[dbo].[Basket] (
			UserId, BasketName, BasketNumber, BasketType,
			PlanName, PlanRootNumber,
			PublishDate, SetDate, AddedAt, isValid
		)
		OUTPUT INSERTED.BasketId
		VALUES (
			@p1, @p2, @p3, @p4,
			@p5, @p6, @p7,
			@p8, GETDATE(), CAST(0 AS BIT)
		)
	`
	if config.Env == "prod" {
		insertQuery = strings.Replace(insertQuery, "[Tender].[dbo]", "[Tender].[logtender]", 1)
	}

	var newID int64
	err = db.QueryRow(insertQuery,
		input.UserId,
		input.BasketName,
		strconv.Itoa(nextNum), // ✅ sequential number
		input.BasketType,
		input.PlanName,
		input.PlanRootNumber,
		publishDate,
		setDate,
	).Scan(&newID)

	if err != nil {
		fmt.Println("❌ Failed to insert basket:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to insert basket")
		return
	}

	fmt.Printf("✅ Basket inserted (id=%d, user=%d, plan=%d, type=%s, num=%d)\n",
		newID, input.UserId, input.PlanRootNumber, input.BasketType, nextNum)

	c.Ctx.Output.SetStatus(http.StatusCreated)
	c.Data["json"] = map[string]interface{}{
		"message":       "Basket created successfully",
		"basket_id":     newID,
		"basket_number": nextNum,
	}
	c.ServeJSON()
}
