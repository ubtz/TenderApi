package controllers

import (
	config "TenderApi/conf"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
)

const (
	fetchMaxAttempts = 3
)

func totalInfoAPIBaseURL() string {
	if value := strings.TrimRight(strings.TrimSpace(os.Getenv("ORDERS_SERVICE_ROOT")), "/"); value != "" {
		return value
	}
	return "http://192.168.4.107:8008"
}

var httpClient = &http.Client{
	Timeout: 90 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
	},
}

type GetTotalInfo struct {
	beego.Controller
}

type OrderJob struct {
	Year  string
	Month int
	Man   int
}

type RefreshSummary struct {
	mu               sync.Mutex
	SuccessfulOrders int                 `json:"successful_orders"`
	FailedOrders     int                 `json:"failed_orders"`
	FailedDetails    []FailedOrderDetail `json:"failed_details,omitempty"`
}

type FailedOrderDetail struct {
	Year   string `json:"year"`
	Month  int    `json:"month"`
	Man    int    `json:"man"`
	RID    string `json:"rid,omitempty"`
	Acct   string `json:"acct,omitempty"`
	Code   string `json:"code,omitempty"`
	Mdocno string `json:"mdocno,omitempty"`
	Pkgno  string `json:"pkgno,omitempty"`
	Dcode  string `json:"dcode,omitempty"`
	Reason string `json:"reason"`
}

func (s *RefreshSummary) Add(success, failed int, details ...FailedOrderDetail) {
	s.mu.Lock()
	s.SuccessfulOrders += success
	s.FailedOrders += failed
	s.FailedDetails = append(s.FailedDetails, details...)
	s.mu.Unlock()
}

type BranchAgg struct {
	Orders int
	Qty    float64
	Amount float64
}

// ================= TOKEN =================
func getToken() (string, error) {
	payload := map[string]string{
		"Username": os.Getenv("ORDERS_API_USERNAME"),
		"Password": os.Getenv("ORDERS_API_PASSWORD"),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(totalInfoAPIBaseURL()+"/v1/token/gtoken", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return "", err
	}

	token, ok := raw["tokendata"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("token not found")
	}

	return token, nil
}

// ================= FETCH =================
func getOrdersRaw(yy, mm, man, token string) (map[string]interface{}, error) {
	var lastErr error

	for attempt := 1; attempt <= fetchMaxAttempts; attempt++ {
		raw, err := getOrdersRawOnce(yy, mm, man, token)
		if err == nil {
			return raw, nil
		}

		lastErr = err
		beego.Warn(
			"getOrdersRaw attempt failed => year:", yy,
			" month:", mm,
			" man:", man,
			" attempt:", attempt,
			" max_attempts:", fetchMaxAttempts,
			" err:", err,
		)

		if attempt < fetchMaxAttempts {
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
		}
	}

	return nil, fmt.Errorf("orders fetch failed after %d attempts: %v", fetchMaxAttempts, lastErr)
}

func getOrdersRawOnce(yy, mm, man, token string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"Yy":    yy,
		"Mm":    mm,
		"Token": token,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", totalInfoAPIBaseURL()+"/v1/orders/list", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("orders API returned status %d: %s", resp.StatusCode, string(rawBody))
	}

	beego.Info("ORDERS RAW RESPONSE:", string(rawBody))

	var raw map[string]interface{}
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return nil, err
	}

	return raw, nil
}

// ================= HELPERS =================
func getString(row map[string]interface{}, key string) string {
	if v, ok := row[key].(string); ok {
		return v
	}
	return ""
}

func parseFloat(val string) float64 {
	f, _ := strconv.ParseFloat(val, 64)
	return f
}

func parseDate(val string) interface{} {
	if val == "" {
		return nil
	}
	t, err := time.Parse("2006/01/02", val)
	if err != nil {
		return nil
	}
	return t
}

func parseDateTime(val string) interface{} {
	if val == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02 15:04:05", val)
	if err != nil {
		return nil
	}
	return t
}

func getSchema() string {
	if config.Env == "prod" {
		return "[Tender].[logtender]"
	}
	return "[Tender].[dbo]"
}

func newFailedOrderDetail(job OrderJob, row map[string]interface{}, reason string) FailedOrderDetail {
	detail := FailedOrderDetail{
		Year:   job.Year,
		Month:  job.Month,
		Man:    job.Man,
		Reason: reason,
	}

	if row == nil {
		return detail
	}

	detail.RID = getString(row, "rid")
	detail.Acct = getString(row, "acct")
	detail.Code = getString(row, "code")
	detail.Mdocno = getString(row, "mdocno")
	detail.Pkgno = getString(row, "pkgno")
	detail.Dcode = getString(row, "dcode")

	return detail
}

func orderKey(row map[string]interface{}) string {
	fields := []string{
		"rid", "acct", "code", "mdocno", "pkgno",
		"measid", "barcode", "zno", "dcode",
		"qty", "qty_nha", "price", "pricesum",
	}

	key := ""
	hasValue := false
	for _, field := range fields {
		value := getString(row, field)
		if value != "" {
			hasValue = true
		}
		key += value + "\x00"
	}

	if !hasValue {
		return ""
	}
	return key
}

// ================= SEQUENTIAL MONTH PROCESSING =================
func processOrderJob(job OrderJob, db *sql.DB, token string, schema string, seenOrders map[string]struct{}, summary *RefreshSummary) {
	yy := job.Year
	mm := job.Month
	man := job.Man

	raw, err := getOrdersRaw(yy, fmt.Sprintf("%d", mm), "", token)
	if err != nil {
		beego.Error("getOrdersRaw error:", err)
		summary.Add(0, 1, newFailedOrderDetail(job, nil, err.Error()))
		return
	}

	records, ok := raw["records"].([]interface{})
	if !ok {
		beego.Error("records not found or invalid for year:", yy, " month:", mm, " man:", man)
		summary.Add(0, 1, newFailedOrderDetail(job, nil, "records not found or invalid"))
		return
	}

	successfulOrders := 0
	failedOrders := 0
	failedDetails := make([]FailedOrderDetail, 0)
	branchMap := make(map[int]*BranchAgg)

	for _, r := range records {
		row, ok := r.(map[string]interface{})
		if !ok {
			failedOrders++
			failedDetails = append(failedDetails, newFailedOrderDetail(job, nil, "row cast failed"))
			beego.Error("row cast failed for year:", yy, " month:", mm, " man:", man)
			continue
		}

		key := orderKey(row)
		if key != "" {
			if _, exists := seenOrders[key]; exists {
				beego.Warn("Skipping repeated monthly order => year:", yy, " month:", mm, " rid:", getString(row, "rid"), " acct:", getString(row, "acct"), " code:", getString(row, "code"))
				continue
			}
			seenOrders[key] = struct{}{}
		}

		qtyVal := parseFloat(getString(row, "qty"))
		priceVal := parseFloat(getString(row, "price"))
		sumVal := parseFloat(getString(row, "pricesum"))

		dcodeStr := getString(row, "dcode")
		branchID := 0
		if dcodeStr != "" {
			branchID, _ = strconv.Atoi(dcodeStr)
		}

		_, err := db.Exec(`
				INSERT INTO `+schema+`.[orders] (
					rid, acct, code,
					cr1id, cr2id, cr3id, cr4id,
					crbrand, crmark,
					cr1name, cr4name, crbrandname, crmarkname,
					mdocno, pkgno,
					ddate, pkgdate, dedate,
					cdate, udate, plandate, techdate,
					measid, mname, usize,
					barcode, zno,
					qty, qty_nha,
					price, pricesum,
					dcode, dname,
					[plan], [tech],
					planurl, techurl,
					man
				)
				VALUES (
					@p1,@p2,@p3,
					@p4,@p5,@p6,@p7,
					@p8,@p9,
					@p10,@p11,@p12,@p13,
					@p14,@p15,
					@p16,@p17,@p18,
					@p19,@p20,@p21,@p22,
					@p23,@p24,@p25,
					@p26,@p27,
					@p28,@p29,
					@p30,@p31,
					@p32,@p33,
					@p34,@p35,
					@p36,@p37,
					@p38
				)
			`,
			getString(row, "rid"),
			getString(row, "acct"),
			getString(row, "code"),

			getString(row, "cr1id"),
			getString(row, "cr2id"),
			getString(row, "cr3id"),
			getString(row, "cr4id"),

			getString(row, "crbrand"),
			getString(row, "crmark"),

			getString(row, "cr1name"),
			getString(row, "cr4name"),
			getString(row, "crbrandname"),
			getString(row, "crmarkname"),

			getString(row, "mdocno"),
			getString(row, "pkgno"),

			parseDate(getString(row, "ddate")),
			parseDate(getString(row, "pkgdate")),
			parseDate(getString(row, "dedate")),

			parseDateTime(getString(row, "cdate")),
			parseDateTime(getString(row, "udate")),
			parseDateTime(getString(row, "plandate")),
			parseDateTime(getString(row, "techdate")),

			getString(row, "measid"),
			getString(row, "mname"),
			getString(row, "usize"),

			getString(row, "barcode"),
			getString(row, "zno"),

			qtyVal,
			nil,

			priceVal,
			sumVal,

			dcodeStr,
			getString(row, "dname"),

			getString(row, "plan"),
			getString(row, "tech"),

			getString(row, "planurl"),
			getString(row, "techurl"),

			man,
		)

		if err != nil {
			failedOrders++
			failedDetails = append(failedDetails, newFailedOrderDetail(job, row, err.Error()))
			beego.Error("Insert order error:", err)
			continue
		}

		successfulOrders++

		if branchID > 0 {
			if _, exists := branchMap[branchID]; !exists {
				branchMap[branchID] = &BranchAgg{}
			}
			branchMap[branchID].Orders++
			branchMap[branchID].Qty += qtyVal
			branchMap[branchID].Amount += sumVal
		} else {
			beego.Warn("Invalid branch_id(dcode) for inserted order => year:", yy, " month:", mm, " man:", man, " dcode:", dcodeStr)
		}
	}

	summary.Add(successfulOrders, failedOrders, failedDetails...)

	for branchID, agg := range branchMap {
		_, err := db.Exec(`
				INSERT INTO `+schema+`.[branch_statistics]
				(branch_id, man, year, month, total_orders, total_qty, total_amount, created_at)
				VALUES(@p1,@p2,@p3,@p4,@p5,@p6,@p7,@p8)
			`,
			branchID,
			man,
			yy,
			mm,
			agg.Orders,
			agg.Qty,
			agg.Amount,
			time.Now(),
		)

		if err != nil {
			beego.Error(
				"Insert statistics error => year:", yy,
				" month:", mm,
				" man:", man,
				" branch:", branchID,
				" err:", err,
			)
			continue
		}

		beego.Info(
			"STAT => year:", yy,
			" month:", mm,
			" man:", man,
			" branch:", branchID,
			" orders:", agg.Orders,
			" qty:", agg.Qty,
			" amount:", agg.Amount,
		)
	}

	beego.Info(
		"JOB DONE => year:", yy,
		" month:", mm,
		" man:", man,
		" success:", successfulOrders,
		" failed:", failedOrders,
		" total_records:", len(records),
		" branches:", len(branchMap),
	)

	if failedOrders > 0 {
		beego.Warn(
			"WARNING: Some inserts failed => year:", yy,
			" month:", mm,
			" man:", man,
			" failed:", failedOrders,
		)
	}
}

// ================= SHARED REFRESH =================
func refreshOrdersAndStatistics() (*RefreshSummary, error) {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	schema := getSchema()

	_, err := db.Exec(`TRUNCATE TABLE ` + schema + `.[orders]`)
	if err != nil {
		return nil, fmt.Errorf("truncate orders failed: %v", err)
	}

	_, err = db.Exec(`
		DELETE FROM ` + schema + `.[branch_statistics]
		WHERE CAST(created_at AS date) = CAST(GETDATE() AS date)
	`)
	if err != nil {
		return nil, fmt.Errorf("delete today's branch_statistics failed: %v", err)
	}

	years := []string{"2025", "2026"}

	token, err := getToken()
	if err != nil {
		return nil, err
	}

	summary := &RefreshSummary{}
	seenOrders := make(map[string]struct{})

	for _, yy := range years {
		processOrderJob(OrderJob{
			Year:  yy,
			Month: 12,
			Man:   0,
		}, db, token, schema, seenOrders, summary)
	}

	beego.Info(
		"Orders + statistics refresh finished. successful_orders:",
		summary.SuccessfulOrders,
		" failed_orders:",
		summary.FailedOrders,
	)

	return summary, nil
}

// ================= MAIN API =================
func (c *GetTotalInfo) GetTotalInfo() {
	summary, err := refreshOrdersAndStatistics()
	if err != nil {
		c.Data["json"] = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
		c.ServeJSON()
		return
	}

	message := "Orders + statistics refreshed successfully"
	if summary.FailedOrders > 0 {
		message = "Orders + statistics refreshed with some failed inserts"
	}

	c.Data["json"] = map[string]interface{}{
		"success":           true,
		"message":           message,
		"successful_orders": summary.SuccessfulOrders,
		"failed_orders":     summary.FailedOrders,
		"failed_details":    summary.FailedDetails,
	}
	c.ServeJSON()
}
