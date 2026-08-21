package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type DeleteTender struct {
	beego.Controller
}

func (c *DeleteTender) Delete() {
	tenderID, err := strconv.Atoi(c.Ctx.Input.Param(":id"))
	if err != nil || tenderID <= 0 {
		c.Ctx.Output.SetStatus(http.StatusBadRequest)
		c.Data["json"] = map[string]string{"error": "Тендерийн ID буруу байна"}
		c.ServeJSON()
		return
	}

	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID == 0 {
		c.Ctx.Output.SetStatus(http.StatusUnauthorized)
		c.Data["json"] = map[string]string{"error": "Нэвтрэх эрх хүчингүй байна"}
		c.ServeJSON()
		return
	}

	db := connectDB(getConfig(config.Env))
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Устгах үйлдлийг эхлүүлж чадсангүй"}
		c.ServeJSON()
		return
	}
	defer tx.Rollback()

	schema := getSchema()
	var createdBy, contractCount, childCount int
	err = tx.QueryRow(`
		SELECT
			t.CreatedBy,
			(SELECT COUNT(1) FROM `+schema+`.[Geree] g WHERE g.TenderId = t.TenderId),
			(SELECT COUNT(1) FROM `+schema+`.[Tender] child WHERE child.RootTenderId = t.TenderId AND ISNULL(child.IsDeleted, 0) = 0)
		FROM `+schema+`.[Tender] t
		WHERE t.TenderId = @p1 AND ISNULL(t.IsDeleted, 0) = 0
	`, tenderID).Scan(&createdBy, &contractCount, &childCount)
	if err == sql.ErrNoRows {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		c.Data["json"] = map[string]string{"error": "Тендер олдсонгүй эсвэл аль хэдийн устсан байна"}
		c.ServeJSON()
		return
	}
	if err != nil {
		log.Printf("DeleteTender dependency query failed: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Тендерийн хамаарлыг шалгаж чадсангүй"}
		c.ServeJSON()
		return
	}

	if claims.Role != "Удирдлага" && createdBy != claims.UserID {
		c.Ctx.Output.SetStatus(http.StatusForbidden)
		c.Data["json"] = map[string]string{"error": "Зөвхөн тендер үүсгэсэн хэрэглэгч устгах боломжтой"}
		c.ServeJSON()
		return
	}
	if contractCount > 0 {
		c.Ctx.Output.SetStatus(http.StatusConflict)
		c.Data["json"] = map[string]string{"error": "Гэрээ үүссэн тендерийг устгах боломжгүй"}
		c.ServeJSON()
		return
	}
	if childCount > 0 {
		c.Ctx.Output.SetStatus(http.StatusConflict)
		c.Data["json"] = map[string]string{"error": "Хувилбар бүхий тендерийг устгах боломжгүй"}
		c.ServeJSON()
		return
	}

	result, err := tx.Exec(`
		UPDATE `+schema+`.[Tender]
		SET IsDeleted = 1
		WHERE TenderId = @p1 AND ISNULL(IsDeleted, 0) = 0
	`, tenderID)
	if err != nil {
		log.Printf("DeleteTender update failed: %v", err)
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Тендер устгахад алдаа гарлаа"}
		c.ServeJSON()
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		c.Ctx.Output.SetStatus(http.StatusNotFound)
		c.Data["json"] = map[string]string{"error": "Тендер олдсонгүй"}
		c.ServeJSON()
		return
	}

	if err := tx.Commit(); err != nil {
		c.Ctx.Output.SetStatus(http.StatusInternalServerError)
		c.Data["json"] = map[string]string{"error": "Тендер устгах үйлдлийг хадгалж чадсангүй"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]interface{}{
		"success":   true,
		"message":   "Тендер амжилттай устгагдлаа",
		"tender_id": tenderID,
	}
	c.ServeJSON()
}
