package controllers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
)

type GetBasketIdsByRootNum struct {
	beego.Controller
}

func (c *GetBasketIdsByRootNum) Get() {
	rootNum := c.Ctx.Input.Param(":rootNum")
	if rootNum == "" {
		c.CustomAbort(http.StatusBadRequest, "rootNum is required")
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	query := `
        SELECT
            t.Basket_Ids,
            t.Тендерийн_дугаар
        FROM [Tender].[dbo].[Tender] t
        WHERE t.RootTenderId IS NULL
          AND t.PlanRootNumber = @rootNum
          AND t.Basket_Ids IS NOT NULL
    `

	if config.Env == "prod" {
		query = `
        SELECT
            t.Basket_Ids,
            t.Тендерийн_дугаар
        FROM [Tender].[logtender].[Tender] t
        WHERE t.RootTenderId IS NULL
          AND t.PlanRootNumber = @rootNum
          AND t.Basket_Ids IS NOT NULL
        `
	}

	rows, err := db.Query(query, sql.Named("rootNum", rootNum))
	if err != nil {
		log.Println("Query error:", err)
		c.CustomAbort(http.StatusInternalServerError, "Query failed")
		return
	}
	defer rows.Close()

	// 🔹 basketId -> tenderNumber
	basketTenderMap := make(map[int]string)
	basketSet := make(map[int]bool)

	for rows.Next() {
		var basketIdsStr sql.NullString
		var tenderNum sql.NullString

		if err := rows.Scan(&basketIdsStr, &tenderNum); err != nil {
			log.Println("Scan error:", err)
			continue
		}

		if !basketIdsStr.Valid {
			continue
		}

		parts := strings.Split(basketIdsStr.String, ",")
		for _, p := range parts {
			id, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil {
				continue
			}

			basketSet[id] = true

			// 🔒 нэг basket → нэг тендер
			if tenderNum.Valid {
				basketTenderMap[id] = tenderNum.String
			}
		}
	}

	basketIds := make([]int, 0, len(basketSet))
	for id := range basketSet {
		basketIds = append(basketIds, id)
	}

	c.Data["json"] = map[string]interface{}{
		"rootNumber":      rootNum,
		"basketIds":       basketIds,
		"basketTenderMap": basketTenderMap,
	}
	c.ServeJSON()
}
