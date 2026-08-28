package controllers

import (
	config "TenderApi/conf"
	"encoding/json"
	"log"
	"strings"
)

type externalRecycleOrder struct {
	PkgNo   string `json:"pkgno"`
	PkgDate string `json:"pkgdate"`
	PkgName string `json:"pkgname"`
	Code    string `json:"code"`
}

type externalRecycleOrderResponse struct {
	Records []externalRecycleOrder `json:"records"`
}

func matchWaitingTenderRecycles(responseBody []byte) {
	var response externalRecycleOrderResponse
	if err := json.Unmarshal(responseBody, &response); err != nil || len(response.Records) == 0 {
		return
	}

	db := connectDB(getConfig(config.Env))
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		log.Printf("Tender recycle matcher transaction failed: %v", err)
		return
	}
	defer tx.Rollback()

	schema := getSchema()
	matchedCodes := int64(0)
	seen := make(map[string]struct{}, len(response.Records))
	for _, order := range response.Records {
		pkgNo := strings.TrimSpace(order.PkgNo)
		pkgName := strings.TrimSpace(order.PkgName)
		code := strings.TrimSpace(order.Code)
		pkgDate, dateErr := parseReannounceDate(order.PkgDate)
		if pkgNo == "" || pkgName == "" || code == "" || dateErr != nil {
			continue
		}

		identity := strings.ToLower(pkgNo) + "|" + pkgDate.Format("2006-01-02") + "|" + strings.ToLower(pkgName) + "|" + strings.ToLower(code)
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}

		result, updateErr := tx.Exec(`
			UPDATE recycleCode
			SET MatchedAt = ISNULL(recycleCode.MatchedAt, SYSDATETIME())
			FROM `+schema+`.[TenderRecycleCodes] recycleCode
			INNER JOIN `+schema+`.[TenderRecyclePackages] recyclePackage
				ON recyclePackage.Id = recycleCode.RecyclePackageId
			INNER JOIN `+schema+`.[TenderRecycleBatch] recycleBatch
				ON recycleBatch.Id = recyclePackage.RecycleBatchId
			WHERE recycleBatch.Status IN (N'WaitingExternal', N'Matching')
			  AND recyclePackage.Status IN (N'Waiting', N'Matched')
			  AND LTRIM(RTRIM(recyclePackage.PkgNo)) = @p1
			  AND recyclePackage.PkgDate = @p2
			  AND LTRIM(RTRIM(recyclePackage.PkgName)) = @p3
			  AND LTRIM(RTRIM(recycleCode.Code)) = @p4
			  AND recycleCode.MatchedAt IS NULL
		`, pkgNo, pkgDate, pkgName, code)
		if updateErr != nil {
			log.Printf("Tender recycle matcher code update failed pkg=%s code=%s: %v", pkgNo, code, updateErr)
			return
		}
		affected, _ := result.RowsAffected()
		matchedCodes += affected
	}

	if matchedCodes == 0 {
		return
	}

	_, err = tx.Exec(`
		UPDATE recyclePackage
		SET Status = N'Matched',
			MatchedAt = ISNULL(recyclePackage.MatchedAt, SYSDATETIME())
		FROM ` + schema + `.[TenderRecyclePackages] recyclePackage
		WHERE recyclePackage.Status = N'Waiting'
		  AND NOT EXISTS (
			SELECT 1
			FROM ` + schema + `.[TenderRecycleCodes] recycleCode
			WHERE recycleCode.RecyclePackageId = recyclePackage.Id
			  AND recycleCode.MatchedAt IS NULL
		  )
	`)
	if err != nil {
		log.Printf("Tender recycle matcher package update failed: %v", err)
		return
	}

	_, err = tx.Exec(`
		UPDATE recycleBatch
		SET Status = N'Matching'
		FROM ` + schema + `.[TenderRecycleBatch] recycleBatch
		WHERE recycleBatch.Status = N'WaitingExternal'
		  AND EXISTS (
			SELECT 1
			FROM ` + schema + `.[TenderRecyclePackages] recyclePackage
			WHERE recyclePackage.RecycleBatchId = recycleBatch.Id
			  AND recyclePackage.Status = N'Matched'
		  )
	`)
	if err != nil {
		log.Printf("Tender recycle matcher batch update failed: %v", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Tender recycle matcher commit failed: %v", err)
		return
	}
	log.Printf("Tender recycle matcher recognized %d waiting code(s)", matchedCodes)
}
