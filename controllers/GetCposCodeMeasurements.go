package controllers

import (
	"crypto/subtle"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type CposCodeMeasurement struct {
	Code              string     `json:"code"`
	MeasurementUnit   *string    `json:"measurementUnit"`
	MeasurementUnitID *int64     `json:"measurementUnitId"`
	EditedByUserID    *int64     `json:"editedByUserId"`
	EditedByUserName  string     `json:"editedByUserName"`
	EditedAt          *time.Time `json:"editedAt"`
}

type CposCodeMeasurementValue struct {
	Code              string `json:"code"`
	MeasurementUnitID *int64 `json:"measurementUnitId"`
}

type GetCposCodeMeasurements struct {
	beego.Controller
}

const cposMeasurementsPassword = "Test123"

func cposCodesTable() string {
	if config.Env == "prod" {
		return "[Tender].[logtender].[CposCodes]"
	}
	return "[Tender].[dbo].[CposCodes]"
}

func cposUsersTable() string {
	if config.Env == "prod" {
		return "[Tender].[logtender].[Users]"
	}
	return "[Tender].[dbo].[Users]"
}

func cposCodeMeasurementSelect() string {
	return fmt.Sprintf(`
	SELECT
		code.[Code],
		code.[MeasurementUnit],
		code.[MeasurementUnitId],
		code.[EditedByUserId],
		code.[EditedAt],
		LTRIM(RTRIM(ISNULL(users.[Ovog], '') + N' ' + ISNULL(users.[Ner], ''))) AS [EditedByUserName]
	FROM %s AS code
	LEFT JOIN %s AS users ON users.[Id] = code.[EditedByUserId]
`, cposCodesTable(), cposUsersTable())
}

type cposMeasurementScanner interface {
	Scan(dest ...interface{}) error
}

func scanCposCodeMeasurement(scanner cposMeasurementScanner) (CposCodeMeasurement, error) {
	var item CposCodeMeasurement
	var measurementUnit sql.NullString
	var measurementUnitID sql.NullInt64
	var editedByUserID sql.NullInt64
	var editedAt sql.NullTime
	var editedByUserName sql.NullString
	if err := scanner.Scan(
		&item.Code,
		&measurementUnit,
		&measurementUnitID,
		&editedByUserID,
		&editedAt,
		&editedByUserName,
	); err != nil {
		return item, err
	}
	if measurementUnit.Valid {
		item.MeasurementUnit = &measurementUnit.String
	}
	if measurementUnitID.Valid {
		item.MeasurementUnitID = &measurementUnitID.Int64
	}
	if editedByUserID.Valid {
		item.EditedByUserID = &editedByUserID.Int64
	}
	if editedAt.Valid {
		item.EditedAt = &editedAt.Time
	}
	if editedByUserName.Valid {
		item.EditedByUserName = editedByUserName.String
	}
	return item, nil
}

func loadCposCodeMeasurement(db *sql.DB, code string) (CposCodeMeasurement, error) {
	return scanCposCodeMeasurement(db.QueryRow(
		cposCodeMeasurementSelect()+` WHERE code.[Code] = @p1`,
		code,
	))
}

func (c *GetCposCodeMeasurements) Get() {
	providedPassword := strings.TrimSpace(c.Ctx.Input.Header("X-API-Password"))
	if len(providedPassword) != len(cposMeasurementsPassword) || subtle.ConstantTimeCompare([]byte(providedPassword), []byte(cposMeasurementsPassword)) != 1 {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]string{"error": "Invalid API password"}
		c.ServeJSON()
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	rows, err := db.Query(cposCodeMeasurementSelect() + ` ORDER BY code.[Code]`)
	if err != nil {
		log.Println("GetCposCodeMeasurements query failed:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to load CPOS code measurements"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	items := make([]CposCodeMeasurementValue, 0)
	for rows.Next() {
		item, err := scanCposCodeMeasurement(rows)
		if err != nil {
			log.Println("GetCposCodeMeasurements row scan failed:", err)
			c.Ctx.Output.SetStatus(http.StatusInternalServerError)
			c.Data["json"] = map[string]string{"error": "Failed to read CPOS code measurements"}
			c.ServeJSON()
			return
		}
		items = append(items, CposCodeMeasurementValue{
			Code:              item.Code,
			MeasurementUnitID: item.MeasurementUnitID,
		})
	}

	if err := rows.Err(); err != nil {
		log.Println("GetCposCodeMeasurements row iteration failed:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to read CPOS code measurements"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = items
	c.ServeJSON()
}

func (c *GetCposCodeMeasurements) GetDetails() {
	db := connectDB(getConfig(config.Env))
	defer db.Close()

	rows, err := db.Query(cposCodeMeasurementSelect() + ` ORDER BY code.[Code]`)
	if err != nil {
		log.Println("GetCposCodeMeasurementDetails query failed:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to load CPOS code measurement details"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	items := make([]CposCodeMeasurement, 0)
	for rows.Next() {
		item, err := scanCposCodeMeasurement(rows)
		if err != nil {
			log.Println("GetCposCodeMeasurementDetails row scan failed:", err)
			c.Ctx.Output.SetStatus(http.StatusInternalServerError)
			c.Data["json"] = map[string]string{"error": "Failed to read CPOS code measurement details"}
			c.ServeJSON()
			return
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		log.Println("GetCposCodeMeasurementDetails row iteration failed:", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Failed to read CPOS code measurement details"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = items
	c.ServeJSON()
}
