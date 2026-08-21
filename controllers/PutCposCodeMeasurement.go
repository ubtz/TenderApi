package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type PutCposCodeMeasurement struct {
	beego.Controller
}

type CposCodeMeasurementInput struct {
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	Mark              string  `json:"mark"`
	Barcode           string  `json:"barcode"`
	SourceUnit        string  `json:"sourceUnit"`
	Rule1No           string  `json:"rule1No"`
	Rule1Name         string  `json:"rule1Name"`
	Rule2             string  `json:"rule2"`
	MeasurementUnit   *string `json:"measurementUnit"`
	MeasurementUnitID *int    `json:"measurementUnitId"`
}

func nullableText(value string) interface{} {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func (c *PutCposCodeMeasurement) Put() {
	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID == 0 {
		c.CustomAbort(http.StatusUnauthorized, "Invalid user session")
		return
	}

	var input CposCodeMeasurementInput
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON")
		return
	}
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	if input.Code == "" || input.Name == "" {
		c.CustomAbort(http.StatusBadRequest, "Code and name are required")
		return
	}

	var measurementUnit interface{}
	var measurementUnitID interface{}
	if input.MeasurementUnit != nil {
		measurementUnit = nullableText(*input.MeasurementUnit)
	}
	if input.MeasurementUnitID != nil {
		if *input.MeasurementUnitID <= 0 {
			c.CustomAbort(http.StatusBadRequest, "measurementUnitId must be greater than zero")
			return
		}
		measurementUnitID = *input.MeasurementUnitID
	}
	if measurementUnit != nil && measurementUnitID == nil {
		c.CustomAbort(http.StatusBadRequest, "A valid measurementUnitId is required")
		return
	}
	log.Printf(
		"PutCposCodeMeasurement request: code=%s measurementUnit=%v measurementUnitId=%v userId=%d",
		input.Code,
		measurementUnit,
		measurementUnitID,
		claims.UserID,
	)

	db := connectDB(getConfig(config.Env))
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Println("PutCposCodeMeasurement transaction failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to save measurement unit")
		return
	}
	defer tx.Rollback()
	cposTable := cposCodesTable()

	var id int64
	err = tx.QueryRow(`
		SELECT [Id]
		FROM `+cposTable+` WITH (UPDLOCK, HOLDLOCK)
		WHERE [Code] = @p1
	`, input.Code).Scan(&id)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		var result sql.Result
		result, err = tx.Exec(`
			INSERT INTO `+cposTable+` (
				[Code], [Name], [Mark], [Barcode], [SourceUnit],
				[Rule1No], [Rule1Name], [Rule2], [MeasurementUnit], [MeasurementUnitId],
				[UpdatedAt], [EditedByUserId], [EditedAt]
			)
			VALUES (
				@p1, @p2, @p3, @p4, @p5,
				@p6, @p7, @p8, @p9, @p10,
				GETDATE(), @p11, GETDATE()
			)
		`,
			input.Code,
			input.Name,
			nullableText(input.Mark),
			nullableText(input.Barcode),
			nullableText(input.SourceUnit),
			nullableText(input.Rule1No),
			nullableText(input.Rule1Name),
			nullableText(input.Rule2),
			measurementUnit,
			measurementUnitID,
			claims.UserID,
		)
		if err == nil {
			rowsAffected, rowsErr := result.RowsAffected()
			log.Printf("PutCposCodeMeasurement insert: code=%s rows=%d rowsError=%v", input.Code, rowsAffected, rowsErr)
		}
	case err != nil:
		log.Println("PutCposCodeMeasurement lookup failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to save measurement unit")
		return
	default:
		var result sql.Result
		result, err = tx.Exec(`
			UPDATE `+cposTable+`
			SET [Name] = @p1,
				[Mark] = @p2,
				[Barcode] = @p3,
				[SourceUnit] = @p4,
				[Rule1No] = @p5,
				[Rule1Name] = @p6,
				[Rule2] = @p7,
				[MeasurementUnit] = @p8,
				[MeasurementUnitId] = @p9,
				[UpdatedAt] = GETDATE(),
				[EditedByUserId] = @p10,
				[EditedAt] = GETDATE()
			WHERE [Id] = @p11
		`,
			input.Name,
			nullableText(input.Mark),
			nullableText(input.Barcode),
			nullableText(input.SourceUnit),
			nullableText(input.Rule1No),
			nullableText(input.Rule1Name),
			nullableText(input.Rule2),
			measurementUnit,
			measurementUnitID,
			claims.UserID,
			id,
		)
		if err == nil {
			rowsAffected, rowsErr := result.RowsAffected()
			log.Printf("PutCposCodeMeasurement update: code=%s id=%d rows=%d rowsError=%v", input.Code, id, rowsAffected, rowsErr)
			if rowsErr != nil || rowsAffected != 1 {
				err = errors.New("measurement update did not affect exactly one row")
			}
		}
	}
	if err != nil {
		log.Println("PutCposCodeMeasurement upsert failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to save measurement unit")
		return
	}

	if err := tx.Commit(); err != nil {
		log.Println("PutCposCodeMeasurement commit failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Failed to save measurement unit")
		return
	}

	item, err := loadCposCodeMeasurement(db, input.Code)
	if err != nil {
		log.Println("PutCposCodeMeasurement response lookup failed:", err)
		c.CustomAbort(http.StatusInternalServerError, "Measurement saved but response could not be loaded")
		return
	}
	log.Printf(
		"PutCposCodeMeasurement saved: code=%s measurementUnit=%v measurementUnitId=%v editedByUserId=%v",
		item.Code,
		item.MeasurementUnit,
		item.MeasurementUnitID,
		item.EditedByUserID,
	)

	c.Data["json"] = item
	c.ServeJSON()
}
