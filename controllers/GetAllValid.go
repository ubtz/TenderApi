package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"fmt"

	"github.com/astaxie/beego"
)

type GetAllValid struct {
	beego.Controller
}

func (c *GetAllValid) GetAllValid() {
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
		bi.State
	FROM [Tender].[dbo].[Basket] b
	JOIN [Tender].[dbo].[BasketItems] bi 
		ON bi.BasketId = b.BasketId
	WHERE b.isValid = 1
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
			bi.crmarkname
			bi.Tech_Tod,
			bi.Tusuv,
			bi.State
		FROM [Tender].[logtender].[Basket] b
		JOIN [Tender].[logtender].[BasketItems] bi 
			ON bi.BasketId = b.BasketId
		WHERE b.isValid = 1
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
			tech_Tod, tusuv                                                                                                      sql.NullString
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
		)
		if err != nil {
			fmt.Println("❌ Row scan error:", err)
			continue
		}

		if _, exists := groupMap[rootNum]; !exists {
			groupMap[rootNum] = &BasketGroupDTO{
				PlanRootNumber: rootNum,
				PlanName:       planName,
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
			groupMap[rootNum].Baskets = append(groupMap[rootNum].Baskets, newBasket)
			basketPtr = &groupMap[rootNum].Baskets[len(groupMap[rootNum].Baskets)-1]
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
		})
	}

	var result []BasketGroupDTO
	for _, g := range groupMap {
		result = append(result, *g)
	}

	c.Data["json"] = result
	c.ServeJSON()
}
