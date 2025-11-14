package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"fmt"
	"time"

	"github.com/astaxie/beego"
)

type GetAll struct {
	beego.Controller
}

type BasketItemDTO struct {
	BasketItemId int             `json:"basket_item_id"`
	Barcode      string          `json:"barcode"`
	Code         string          `json:"code"`
	DName        string          `json:"dname"`
	Price        float64         `json:"price"`
	Qty          float64         `json:"qty"`
	PriceSum     float64         `json:"pricesum"`
	Zno          string          `json:"zno"`
	Cr4name      string          `json:"cr4name"`
	Crmarkname   string          `json:"crmarkname"`
	Usize        string          `json:"usize"`
	Mname        string          `json:"mname"`
	State        uint8           `json:"state"`
	Tech_Tod     *sql.NullString `json:"tech_tod"`
	Tusuv        *sql.NullString `json:"tusuv"`
	PkgNo        *sql.NullString `json:"pkgno"`
	TechDate     *sql.NullString `json:"techdate"`
	TechUrl      *sql.NullString `json:"techurl"`
	PlanDate     *sql.NullString `json:"plandate"`
	PlanUrl      *sql.NullString `json:"planurl"`
	PkgDate      *sql.NullString `json:"pkgdate"`
	Key          *sql.NullString `json:"key"`
}

type BasketDTO struct {
	BasketId       int             `json:"basket_id"`
	UserId         int             `json:"user_id"`
	AddedAt        string          `json:"added_at"`
	BasketName     string          `json:"basket_name"`
	BasketNumber   string          `json:"basket_number"`
	BasketType     string          `json:"basket_type"`
	PublishDate    string          `json:"publish_date"`
	PlanName       string          `json:"plan_name"`
	PlanRootNumber string          `json:"plan_root_number"`
	SetDate        string          `json:"set_date"`
	IsValid        bool            `json:"is_valid"`
	Items          []BasketItemDTO `json:"items"`
}

type BasketGroupDTO struct {
	PlanRootNumber string      `json:"plan_root_number"`
	PlanName       string      `json:"plan_name"`
	UserId         int         `json:"user_id"`
	Baskets        []BasketDTO `json:"baskets"`
}

func (c *GetAll) GetAll() {
	fmt.Println("📥 GetBasket endpoint hit")

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
		SELECT 
			b.PlanRootNumber,
			b.BasketId,
			b.UserId,
			b.AddedAt,
			b.BasketName,
			b.BasketNumber,
			b.BasketType,
			b.PublishDate,
			b.PlanName,
			b.SetDate,
			b.isValid,
			bi.BasketItemId,
			bi.barcode,
			bi.code,
			bi.dname,
			bi.price,
			bi.qty,
			bi.pricesum,
			bi.mname,
			bi.usize,
			bi.zno,
			bi.cr4name,
			bi.crmarkname,
			bi.Tech_Tod,
			bi.Tusuv,
			bi.State,
			bi.pkgno,
			bi.techdate,
			bi.techurl,
			bi.plandate,
			bi.planurl,
			bi.pkgdate,
			bi.[key]
		FROM [Tender].[dbo].[Basket] b
		JOIN [Tender].[dbo].[BasketItems] bi 
			ON bi.BasketId = b.BasketId
		ORDER BY 
			b.PlanRootNumber ASC, 
			b.BasketType ASC,
			bi.BasketItemId ASC;`

	if config.Env == "prod" {
		query = `
		SELECT 
			b.PlanRootNumber,
			b.BasketId,
			b.UserId,
			b.AddedAt,
			b.BasketName,
			b.BasketNumber,
			b.BasketType,
			b.PublishDate,
			b.PlanName,
			b.SetDate,
			b.isValid,
			bi.BasketItemId,
			bi.barcode,
			bi.code,
			bi.dname,
			bi.price,
			bi.qty,
			bi.pricesum,
			bi.mname,
			bi.usize,
			bi.zno,
			bi.cr4name,
			bi.crmarkname,
			bi.Tech_Tod,
			bi.Tusuv,
			bi.State,
			bi.pkgno,
			bi.techdate,
			bi.techurl,
			bi.plandate,
			bi.planurl,
			bi.pkgdate,
			bi.[key]
		FROM [Tender].[logtender].[Basket] b
		JOIN [Tender].[logtender].[BasketItems] bi 
			ON bi.BasketId = b.BasketId
		ORDER BY 
			b.PlanRootNumber ASC,
			b.BasketType ASC,
			bi.BasketItemId ASC;`
	}

	rows, err := db.Query(query)
	if err != nil {
		fmt.Println("❌ DB query error:", err)
		c.Ctx.Output.SetStatus(500)
		c.Data["json"] = map[string]string{"error": "Failed to fetch baskets"}
		c.ServeJSON()
		return
	}
	defer rows.Close()

	groupMap := make(map[string]*BasketGroupDTO)
	basketMap := make(map[int]*BasketDTO)

	for rows.Next() {
		var (
			rootNum, basketName, basketNumber, basketType, planName, barcode, code, dname, zno, c4name, crmarkname, mname, usize string
			tech_Tod, tusuv, pkgno, techdate, techurl, plandate, planurl, pkgdate, key                                           sql.NullString
			addedAt, publishDate, setDate                                                                                        sql.NullTime
			basketId, userId, basketItemId                                                                                       int
			price, qty, pricesum                                                                                                 float64
			state                                                                                                                uint8
			isValid                                                                                                              bool
		)

		err := rows.Scan(
			&rootNum,
			&basketId,
			&userId,
			&addedAt,
			&basketName,
			&basketNumber,
			&basketType,
			&publishDate,
			&planName,
			&setDate,
			&isValid,
			&basketItemId,
			&barcode,
			&code,
			&dname,
			&price,
			&qty,
			&pricesum,
			&mname,
			&usize,
			&zno,
			&c4name,
			&crmarkname,
			&tech_Tod,
			&tusuv,
			&state,
			&pkgno,
			&techdate,
			&techurl,
			&plandate,
			&planurl,
			&pkgdate,
			&key,
		)
		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		groupKey := fmt.Sprintf("%s_%d", rootNum, userId)

		if _, exists := groupMap[groupKey]; !exists {
			groupMap[groupKey] = &BasketGroupDTO{
				PlanRootNumber: rootNum,
				PlanName:       planName,
				UserId:         userId,
				Baskets:        []BasketDTO{},
			}
		}

		basketPtr, exists := basketMap[basketId]
		if !exists {
			newBasket := BasketDTO{
				BasketId:       basketId,
				UserId:         userId,
				AddedAt:        formatTime(addedAt),
				BasketName:     basketName,
				BasketNumber:   basketNumber,
				BasketType:     basketType,
				PublishDate:    formatTime(publishDate),
				PlanName:       planName,
				PlanRootNumber: rootNum,
				SetDate:        formatTime(setDate),
				IsValid:        isValid,
				Items:          []BasketItemDTO{},
			}
			groupMap[groupKey].Baskets = append(groupMap[groupKey].Baskets, newBasket)
			basketPtr = &groupMap[groupKey].Baskets[len(groupMap[groupKey].Baskets)-1]
			basketMap[basketId] = basketPtr
		}

		basketPtr.Items = append(basketPtr.Items, BasketItemDTO{
			BasketItemId: basketItemId,
			Barcode:      barcode,
			Code:         code,
			DName:        dname,
			Price:        price,
			Qty:          qty,
			PriceSum:     pricesum,
			Mname:        mname,
			Usize:        usize,
			Zno:          zno,
			Cr4name:      c4name,
			Crmarkname:   crmarkname,
			State:        state,
			Tech_Tod:     &tech_Tod,
			Tusuv:        &tusuv,
			PkgNo:        &pkgno,
			TechDate:     &techdate,
			TechUrl:      &techurl,
			PlanDate:     &plandate,
			PlanUrl:      &planurl,
			PkgDate:      &pkgdate,
			Key:          &key,
		})
	}

	var result []BasketGroupDTO
	for _, g := range groupMap {
		result = append(result, *g)
	}

	c.Data["json"] = result
	c.ServeJSON()
}

func formatTime(t sql.NullTime) string {
	if !t.Valid {
		return ""
	}
	return t.Time.Format(time.RFC3339)
}
