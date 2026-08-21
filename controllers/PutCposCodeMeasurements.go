package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

const maxCposMeasurementBatchSize = 2000

type PutCposCodeMeasurements struct {
	beego.Controller
}

type CposCodeMeasurementsInput struct {
	Items []CposCodeMeasurementInput `json:"items"`
}

func prepareCposMeasurementInput(input *CposCodeMeasurementInput) (interface{}, interface{}, error) {
	input.Code = strings.TrimSpace(input.Code)
	input.Name = strings.TrimSpace(input.Name)
	if input.Code == "" || input.Name == "" {
		return nil, nil, errors.New("code and name are required")
	}

	var measurementUnit interface{}
	if input.MeasurementUnit != nil {
		measurementUnit = nullableText(*input.MeasurementUnit)
	}
	if measurementUnit == nil {
		return nil, nil, errors.New("measurementUnit is required")
	}
	if input.MeasurementUnitID == nil || *input.MeasurementUnitID <= 0 {
		return nil, nil, errors.New("a valid measurementUnitId is required")
	}
	return measurementUnit, *input.MeasurementUnitID, nil
}

func upsertCposMeasurement(tx *sql.Tx, table string, input CposCodeMeasurementInput, userID int) (bool, error) {
	measurementUnit, measurementUnitID, err := prepareCposMeasurementInput(&input)
	if err != nil {
		return false, err
	}

	var id int64
	err = tx.QueryRow(`
		SELECT [Id]
		FROM `+table+` WITH (UPDLOCK, HOLDLOCK)
		WHERE [Code] = @p1
	`, input.Code).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		_, err = tx.Exec(`
			INSERT INTO `+table+` (
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
			userID,
		)
		return true, err
	}
	if err != nil {
		return false, err
	}

	result, err := tx.Exec(`
		UPDATE `+table+`
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
		userID,
		id,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected != 1 {
		return false, errors.New("measurement update did not affect exactly one row")
	}
	return false, nil
}

func (c *PutCposCodeMeasurements) Put() {
	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID == 0 {
		c.CustomAbort(http.StatusUnauthorized, "Invalid user session")
		return
	}

	var input CposCodeMeasurementsInput
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil {
		c.CustomAbort(http.StatusBadRequest, "Invalid JSON")
		return
	}
	if len(input.Items) == 0 {
		c.CustomAbort(http.StatusBadRequest, "At least one item is required")
		return
	}
	if len(input.Items) > maxCposMeasurementBatchSize {
		c.CustomAbort(http.StatusBadRequest, fmt.Sprintf("A maximum of %d items is allowed", maxCposMeasurementBatchSize))
		return
	}

	itemsByCode := make(map[string]CposCodeMeasurementInput, len(input.Items))
	for index := range input.Items {
		item := input.Items[index]
		item.Code = strings.TrimSpace(item.Code)
		if _, _, err := prepareCposMeasurementInput(&item); err != nil {
			c.CustomAbort(http.StatusBadRequest, fmt.Sprintf("Item %d: %s", index+1, err.Error()))
			return
		}
		itemsByCode[item.Code] = item
	}

	items := make([]CposCodeMeasurementInput, 0, len(itemsByCode))
	for _, item := range itemsByCode {
		items = append(items, item)
	}
	sort.Slice(items, func(first, second int) bool { return items[first].Code < items[second].Code })

	db := connectDB(getConfig(config.Env))
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to start measurement import")
		return
	}
	defer tx.Rollback()

	inserted := 0
	updated := 0
	for _, item := range items {
		wasInserted, upsertErr := upsertCposMeasurement(tx, cposCodesTable(), item, claims.UserID)
		if upsertErr != nil {
			log.Printf("PutCposCodeMeasurements failed: code=%s error=%v", item.Code, upsertErr)
			c.CustomAbort(http.StatusInternalServerError, fmt.Sprintf("Failed to save code %s", item.Code))
			return
		}
		if wasInserted {
			inserted++
		} else {
			updated++
		}
	}
	if err := tx.Commit(); err != nil {
		c.CustomAbort(http.StatusInternalServerError, "Failed to commit measurement import")
		return
	}

	savedItems := make([]CposCodeMeasurement, 0, len(items))
	for _, item := range items {
		saved, loadErr := loadCposCodeMeasurement(db, item.Code)
		if loadErr != nil {
			log.Printf("PutCposCodeMeasurements response lookup failed: code=%s error=%v", item.Code, loadErr)
			continue
		}
		savedItems = append(savedItems, saved)
	}

	c.Data["json"] = map[string]interface{}{
		"total":             len(items),
		"inserted":          inserted,
		"updated":           updated,
		"duplicatesRemoved": len(input.Items) - len(items),
		"items":             savedItems,
	}
	c.ServeJSON()
}
