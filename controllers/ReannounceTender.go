package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

const (
	reannounceDecisionSuccessful = 1
	reannounceDecisionRecycle    = 2
)

type ReannounceTender struct {
	beego.Controller
}

type ReannounceTenderInput struct {
	TenderID     int                       `json:"tenderId"`
	TenderNumber string                    `json:"tenderNumber"`
	TenderName   string                    `json:"tenderName"`
	Tailbar      string                    `json:"tailbar"`
	Packages     []ReannounceTenderPackage `json:"packages"`
}

type ReannounceTenderPackage struct {
	PkgNo   string                 `json:"pkgno"`
	PkgDate string                 `json:"pkgdate"`
	PkgName string                 `json:"pkgname"`
	Codes   []ReannounceTenderCode `json:"codes"`
}

type ReannounceTenderCode struct {
	Code   string `json:"code"`
	Status string `json:"status"`
}

type normalizedReannouncePackage struct {
	PkgNo   string
	PkgDate time.Time
	PkgName string
	Codes   []ReannounceTenderCode
}

type reannounceOutboundPayload struct {
	Tailbar  string                      `json:"tailbar"`
	Packages []reannounceOutboundPackage `json:"packages"`
}

type reannounceOutboundPackage struct {
	PkgNo   string                   `json:"pkgno"`
	PkgDate string                   `json:"pkgdate"`
	PkgName string                   `json:"pkgname"`
	Codes   []reannounceOutboundCode `json:"codes"`
}

type reannounceOutboundCode struct {
	Code string `json:"code"`
}

func (c *ReannounceTender) Post() {
	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID <= 0 {
		c.reannounceError(http.StatusUnauthorized, "Нэвтрэх эрх хүчингүй байна")
		return
	}

	var input ReannounceTenderInput
	if err := json.Unmarshal(c.Ctx.Input.RequestBody, &input); err != nil {
		c.reannounceError(http.StatusBadRequest, "Хүсэлтийн JSON бүтэц буруу байна")
		return
	}
	if input.TenderID <= 0 {
		input.TenderID, _ = strconv.Atoi(strings.TrimSpace(c.Ctx.Input.Param(":id")))
	}
	archiveCurrent := !strings.EqualFold(strings.TrimSpace(c.GetString("archive")), "false")

	packages, recyclePackageCount, recycleCodeCount, err := normalizeReannounceInput(input)
	if err != nil {
		c.reannounceError(http.StatusBadRequest, err.Error())
		return
	}

	normalizedPayload, err := buildReannouncePayload(input, packages)
	if err != nil {
		log.Printf("ReannounceTender payload marshal failed: %v", err)
		c.reannounceError(http.StatusInternalServerError, "Хүсэлтийг хадгалахад бэлтгэж чадсангүй")
		return
	}

	db := connectDB(getConfig(config.Env))
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		c.reannounceError(http.StatusInternalServerError, "Дахин зарлах үйлдлийг эхлүүлж чадсангүй")
		return
	}
	defer tx.Rollback()

	if err := lockTenderForReannounce(tx, input.TenderID); err != nil {
		log.Printf("ReannounceTender lock failed tender=%d: %v", input.TenderID, err)
		c.reannounceError(http.StatusConflict, "Энэ тендер дээр өөр үйлдэл хийгдэж байна. Дахин оролдоно уу")
		return
	}

	schema := getSchema()
	var createdBy, versionNo, contractCount, suspended int
	var lifecycleStatus string
	var isCurrent bool
	err = tx.QueryRow(`
		SELECT
			t.CreatedBy,
			ISNULL(t.VersionNo, 1),
			ISNULL(t.IsCurrent, 1),
			ISNULL(t.LifecycleStatus, N'Active'),
			CASE WHEN CONVERT(NVARCHAR(10), t.[Түтгэлзүүлсэн_огноо], 23) = '1900-01-01' THEN 0 ELSE 1 END,
			(SELECT COUNT(1) FROM `+schema+`.[Geree] g WHERE g.TenderId = t.TenderId)
		FROM `+schema+`.[Tender] t WITH (UPDLOCK, HOLDLOCK)
		WHERE t.TenderId = @p1 AND ISNULL(t.IsDeleted, 0) = 0
	`, input.TenderID).Scan(&createdBy, &versionNo, &isCurrent, &lifecycleStatus, &suspended, &contractCount)
	if errors.Is(err, sql.ErrNoRows) {
		c.reannounceError(http.StatusNotFound, "Тендер олдсонгүй")
		return
	}
	if err != nil {
		log.Printf("ReannounceTender tender query failed tender=%d: %v", input.TenderID, err)
		c.reannounceError(http.StatusInternalServerError, "Тендерийн мэдээллийг шалгаж чадсангүй")
		return
	}
	if createdBy != claims.UserID {
		c.reannounceError(http.StatusForbidden, "Зөвхөн тендер үүсгэсэн хэрэглэгч дахин зарлах боломжтой")
		return
	}
	if !isCurrent || lifecycleStatus != "Active" {
		c.reannounceError(http.StatusConflict, "Зөвхөн идэвхтэй одоогийн хувилбарыг дахин зарлах боломжтой")
		return
	}
	if suspended == 1 {
		c.reannounceError(http.StatusConflict, "Түдгэлзүүлсэн тендерийг дахин зарлах боломжгүй")
		return
	}
	if contractCount > 0 {
		c.reannounceError(http.StatusConflict, "Гэрээ үүссэн тендерийг дахин зарлах боломжгүй")
		return
	}

	var openBatchCount int
	err = tx.QueryRow(`
		SELECT COUNT(1)
		FROM `+schema+`.[TenderRecycleBatch]
		WHERE SourceTenderId = @p1
		  AND Status IN (N'WaitingExternal', N'Matching')
	`, input.TenderID).Scan(&openBatchCount)
	if err != nil {
		log.Printf("ReannounceTender batch check failed tender=%d: %v", input.TenderID, err)
		c.reannounceError(http.StatusInternalServerError, "Өмнөх дахин зарлалтыг шалгаж чадсангүй")
		return
	}
	if openBatchCount > 0 {
		c.reannounceError(http.StatusConflict, "Энэ тендерийн дахин зарлалт гадаад системийн мэдээлэл хүлээж байна")
		return
	}

	var batchID int64
	err = tx.QueryRow(`
		INSERT INTO `+schema+`.[TenderRecycleBatch]
			(SourceTenderId, SourceVersionNo, Status, CreatedByUserId)
		OUTPUT INSERTED.Id
		VALUES (@p1, @p2, N'WaitingExternal', @p3)
	`, input.TenderID, versionNo, claims.UserID).Scan(&batchID)
	if err != nil {
		log.Printf("ReannounceTender batch insert failed tender=%d: %v", input.TenderID, err)
		c.reannounceError(http.StatusInternalServerError, "Дахин зарлалтын бүртгэл үүсгэж чадсангүй")
		return
	}

	for _, pkg := range packages {
		var recyclePackageID int64
		hasRecycle := packageHasRecycle(pkg)
		if hasRecycle {
			err = tx.QueryRow(`
				INSERT INTO `+schema+`.[TenderRecyclePackages]
					(RecycleBatchId, PkgNo, PkgDate, PkgName, Status)
				OUTPUT INSERTED.Id
				VALUES (@p1, @p2, @p3, @p4, N'Waiting')
			`, batchID, pkg.PkgNo, pkg.PkgDate, pkg.PkgName).Scan(&recyclePackageID)
			if err != nil {
				log.Printf("ReannounceTender package insert failed batch=%d pkg=%s: %v", batchID, pkg.PkgNo, err)
				c.reannounceError(http.StatusInternalServerError, "Дахин зарлах багцыг хадгалж чадсангүй")
				return
			}
		}

		for _, code := range pkg.Codes {
			decision := reannounceDecisionRecycle

			_, err = tx.Exec(`
				INSERT INTO `+schema+`.[TenderVersionItems]
					(TenderId, BasketId, PkgNo, PkgDate, PkgName, Code, Decision, DecidedByUserId, DecidedAt)
				VALUES (@p1, NULL, @p2, @p3, @p4, @p5, @p6, @p7, SYSDATETIME())
			`, input.TenderID, pkg.PkgNo, pkg.PkgDate, pkg.PkgName, code.Code, decision, claims.UserID)
			if err != nil {
				log.Printf("ReannounceTender result insert failed tender=%d pkg=%s code=%s: %v", input.TenderID, pkg.PkgNo, code.Code, err)
				c.reannounceError(http.StatusInternalServerError, "Тендерийн үр дүнг хадгалж чадсангүй")
				return
			}

			_, err = tx.Exec(`
				INSERT INTO `+schema+`.[TenderRecycleCodes] (RecyclePackageId, Code)
				VALUES (@p1, @p2)
			`, recyclePackageID, code.Code)
			if err != nil {
				log.Printf("ReannounceTender code insert failed package=%d code=%s: %v", recyclePackageID, code.Code, err)
				c.reannounceError(http.StatusInternalServerError, "Дахин зарлах кодыг хадгалж чадсангүй")
				return
			}
		}
	}

	_, err = tx.Exec(`
		INSERT INTO `+schema+`.[TenderRecycleOutbox]
			(RecycleBatchId, Payload, Status)
		VALUES (@p1, @p2, N'Pending')
	`, batchID, string(normalizedPayload))
	if err != nil {
		log.Printf("ReannounceTender outbox insert failed batch=%d: %v", batchID, err)
		c.reannounceError(http.StatusInternalServerError, "Гадаад системд илгээх хүсэлтийг хадгалж чадсангүй")
		return
	}

	if archiveCurrent {
		result, updateErr := tx.Exec(`
			UPDATE `+schema+`.[Tender]
			SET IsCurrent = 0,
				LifecycleStatus = N'WaitingExternal',
				ArchivedAt = SYSDATETIME()
			WHERE TenderId = @p1
			  AND ISNULL(IsDeleted, 0) = 0
			  AND IsCurrent = 1
			  AND LifecycleStatus = N'Active'
		`, input.TenderID)
		if updateErr != nil {
			log.Printf("ReannounceTender archive failed tender=%d: %v", input.TenderID, updateErr)
			c.reannounceError(http.StatusInternalServerError, "Тендерийг архивлаж чадсангүй")
			return
		}
		affected, _ := result.RowsAffected()
		if affected != 1 {
			c.reannounceError(http.StatusConflict, "Тендерийн төлөв өөрчлөгдсөн байна. Мэдээллээ шинэчилнэ үү")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("ReannounceTender commit failed tender=%d batch=%d: %v", input.TenderID, batchID, err)
		c.reannounceError(http.StatusInternalServerError, "Дахин зарлах мэдээллийг баталгаажуулж чадсангүй")
		return
	}

	log.Printf("ReannounceTender created tender=%d version=%d batch=%d packages=%d codes=%d user=%d", input.TenderID, versionNo, batchID, recyclePackageCount, recycleCodeCount, claims.UserID)
	c.Ctx.Output.SetStatus(http.StatusCreated)
	messageText := "Сонгосон захиалгууд дахин зарлах дараалалд орлоо. Үлдсэн захиалгаар гэрээ үүсгэх боломжтой"
	if archiveCurrent {
		messageText = "Тендер архивлагдаж, гадаад системийн мэдээлэл хүлээж байна"
	}
	c.Data["json"] = map[string]interface{}{
		"success":              true,
		"message":              messageText,
		"archived":             archiveCurrent,
		"recycleBatchId":       batchID,
		"status":               "WaitingExternal",
		"recycledPackageCount": recyclePackageCount,
		"recycledCodeCount":    recycleCodeCount,
	}
	c.ServeJSON()
}

func normalizeReannounceInput(input ReannounceTenderInput) ([]normalizedReannouncePackage, int, int, error) {
	if input.TenderID <= 0 {
		return nil, 0, 0, errors.New("тендерийн ID буруу байна")
	}
	if len(input.Packages) == 0 {
		return nil, 0, 0, errors.New("дор хаяж нэг багц шаардлагатай")
	}
	if len(strings.TrimSpace(input.Tailbar)) > 1000 {
		return nil, 0, 0, errors.New("тайлбар хэт урт байна")
	}

	packages := make([]normalizedReannouncePackage, 0, len(input.Packages))
	packageKeys := make(map[string]struct{}, len(input.Packages))
	recyclePackageCount := 0
	recycleCodeCount := 0
	for packageIndex, pkg := range input.Packages {
		pkgNo := strings.TrimSpace(pkg.PkgNo)
		pkgName := strings.TrimSpace(pkg.PkgName)
		if pkgNo == "" || pkgName == "" || strings.TrimSpace(pkg.PkgDate) == "" {
			return nil, 0, 0, fmt.Errorf("%d-р багцын дугаар, нэр, огноо бүрэн биш байна", packageIndex+1)
		}
		if len(pkgNo) > 100 || len(pkgName) > 150 {
			return nil, 0, 0, fmt.Errorf("%d-р багцын дугаар эсвэл нэр хэт урт байна", packageIndex+1)
		}
		pkgDate, err := parseReannounceDate(pkg.PkgDate)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("%d-р багцын огноо буруу байна", packageIndex+1)
		}
		packageKey := strings.ToLower(pkgNo) + "|" + pkgDate.Format("2006-01-02") + "|" + strings.ToLower(pkgName)
		if _, exists := packageKeys[packageKey]; exists {
			return nil, 0, 0, fmt.Errorf("%d-р багц давхардсан байна", packageIndex+1)
		}
		packageKeys[packageKey] = struct{}{}
		if len(pkg.Codes) == 0 {
			return nil, 0, 0, fmt.Errorf("%d-р багц кодгүй байна", packageIndex+1)
		}

		codes := make([]ReannounceTenderCode, 0, len(pkg.Codes))
		codeKeys := make(map[string]struct{}, len(pkg.Codes))
		packageRecycleCount := 0
		for codeIndex, item := range pkg.Codes {
			code := strings.TrimSpace(item.Code)
			status := strings.ToLower(strings.TrimSpace(item.Status))
			if status == "" {
				status = "recycle"
			}
			if code == "" {
				return nil, 0, 0, fmt.Errorf("%d-р багцын %d-р код хоосон байна", packageIndex+1, codeIndex+1)
			}
			if len(code) > 50 {
				return nil, 0, 0, fmt.Errorf("%d-р багцын %d-р код хэт урт байна", packageIndex+1, codeIndex+1)
			}
			if status != "recycle" {
				return nil, 0, 0, fmt.Errorf("%d-р багцын %s кодын төлөв буруу байна", packageIndex+1, code)
			}
			codeKey := strings.ToLower(code)
			if _, exists := codeKeys[codeKey]; exists {
				return nil, 0, 0, fmt.Errorf("%d-р багцын %s код давхардсан байна", packageIndex+1, code)
			}
			codeKeys[codeKey] = struct{}{}
			codes = append(codes, ReannounceTenderCode{Code: code, Status: status})
			if status == "recycle" {
				packageRecycleCount++
				recycleCodeCount++
			}
		}
		if packageRecycleCount > 0 {
			recyclePackageCount++
		}
		packages = append(packages, normalizedReannouncePackage{
			PkgNo:   pkgNo,
			PkgDate: pkgDate,
			PkgName: pkgName,
			Codes:   codes,
		})
	}
	if recycleCodeCount == 0 {
		return nil, 0, 0, errors.New("дахин зарлах дор хаяж нэг код сонгоно уу")
	}
	return packages, recyclePackageCount, recycleCodeCount, nil
}

func parseReannounceDate(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	for _, layout := range []string{"2006/01/02", "2006-01-02", time.RFC3339, "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("invalid date")
}

func packageHasRecycle(pkg normalizedReannouncePackage) bool {
	for _, code := range pkg.Codes {
		if code.Status == "recycle" {
			return true
		}
	}
	return false
}

func buildReannouncePayload(input ReannounceTenderInput, packages []normalizedReannouncePackage) ([]byte, error) {
	normalized := reannounceOutboundPayload{
		Tailbar:  strings.TrimSpace(input.Tailbar),
		Packages: make([]reannounceOutboundPackage, 0, len(packages)),
	}
	for _, pkg := range packages {
		codes := make([]reannounceOutboundCode, 0, len(pkg.Codes))
		for _, code := range pkg.Codes {
			codes = append(codes, reannounceOutboundCode{Code: code.Code})
		}
		normalized.Packages = append(normalized.Packages, reannounceOutboundPackage{
			PkgNo:   pkg.PkgNo,
			PkgDate: pkg.PkgDate.Format("2006/01/02"),
			PkgName: pkg.PkgName,
			Codes:   codes,
		})
	}
	return json.Marshal(normalized)
}

func lockTenderForReannounce(tx *sql.Tx, tenderID int) error {
	var lockResult int
	resource := fmt.Sprintf("TenderReannounce:%d", tenderID)
	err := tx.QueryRow(`
		DECLARE @result INT;
		EXEC @result = sp_getapplock
			@Resource = @p1,
			@LockMode = 'Exclusive',
			@LockOwner = 'Transaction',
			@LockTimeout = 5000;
		SELECT @result;
	`, resource).Scan(&lockResult)
	if err != nil {
		return err
	}
	if lockResult < 0 {
		return fmt.Errorf("sp_getapplock returned %d", lockResult)
	}
	return nil
}

func (c *ReannounceTender) reannounceError(status int, message string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = map[string]interface{}{
		"success": false,
		"error":   message,
	}
	c.ServeJSON()
}
