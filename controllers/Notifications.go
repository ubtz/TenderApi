package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
)

type Notifications struct {
	beego.Controller
}

type NotificationRecord struct {
	ID         int64          `json:"id"`
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Message    sql.NullString `json:"message"`
	EntityType sql.NullString `json:"entityType"`
	EntityID   sql.NullInt64  `json:"entityId"`
	Link       sql.NullString `json:"link"`
	IsRead     bool           `json:"isRead"`
	CreatedAt  time.Time      `json:"createdAt"`
	ReadAt     sql.NullTime   `json:"readAt"`
}

func notificationTable() string {
	return getSchema() + ".[Notifications]"
}

func CreateNotification(db *sql.DB, userID int, notificationType, title, notificationMessage, entityType string, entityID int64, link string) error {
	if userID <= 0 || strings.TrimSpace(title) == "" {
		return nil
	}
	_, err := db.Exec(`
		INSERT INTO `+notificationTable()+`
			(UserId, Type, Title, Message, EntityType, EntityId, Link, IsRead, CreatedAt)
		VALUES (@p1, @p2, @p3, NULLIF(@p4, ''), NULLIF(@p5, ''), NULLIF(@p6, 0), NULLIF(@p7, ''), 0, GETDATE())
	`, userID, notificationType, title, notificationMessage, entityType, entityID, link)
	return err
}

func createNotificationSafe(db *sql.DB, userID int, notificationType, title, notificationMessage, entityType string, entityID int64, link string) {
	if err := CreateNotification(db, userID, notificationType, title, notificationMessage, entityType, entityID, link); err != nil {
		log.Printf("Notification insert failed for user %d: %v", userID, err)
	}
}

func notificationClaims(c *Notifications) (*Claims, bool) {
	claims, err := ClaimsForController(&c.Controller)
	if err != nil || claims.UserID <= 0 {
		c.CustomAbort(http.StatusUnauthorized, "Invalid or missing token")
		return nil, false
	}
	return claims, true
}

func (c *Notifications) GetAll() {
	claims, ok := notificationClaims(c)
	if !ok {
		return
	}
	db := connectDB(getConfig(config.Env))
	defer db.Close()

	rows, err := db.Query(`
		SELECT TOP (50) Id, Type, Title, Message, EntityType, EntityId, Link, IsRead, CreatedAt, ReadAt
		FROM `+notificationTable()+`
		WHERE UserId = @p1
		ORDER BY CreatedAt DESC, Id DESC
	`, claims.UserID)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	records := make([]NotificationRecord, 0)
	for rows.Next() {
		var record NotificationRecord
		if err := rows.Scan(&record.ID, &record.Type, &record.Title, &record.Message, &record.EntityType, &record.EntityID, &record.Link, &record.IsRead, &record.CreatedAt, &record.ReadAt); err != nil {
			c.CustomAbort(http.StatusInternalServerError, err.Error())
			return
		}
		records = append(records, record)
	}
	c.Data["json"] = records
	c.ServeJSON()
}

func (c *Notifications) GetCount() {
	claims, ok := notificationClaims(c)
	if !ok {
		return
	}
	db := connectDB(getConfig(config.Env))
	defer db.Close()

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM `+notificationTable()+` WHERE UserId = @p1 AND IsRead = 0`, claims.UserID).Scan(&count); err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}
	c.Data["json"] = map[string]int{"count": count}
	c.ServeJSON()
}

func (c *Notifications) MarkRead() {
	claims, ok := notificationClaims(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Ctx.Input.Param(":id"), 10, 64)
	if err != nil || id <= 0 {
		c.CustomAbort(http.StatusBadRequest, "Invalid notification id")
		return
	}
	db := connectDB(getConfig(config.Env))
	defer db.Close()

	result, err := db.Exec(`
		UPDATE `+notificationTable()+`
		SET IsRead = 1, ReadAt = COALESCE(ReadAt, GETDATE())
		WHERE Id = @p1 AND UserId = @p2
	`, id, claims.UserID)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}
	updated, _ := result.RowsAffected()
	if updated == 0 {
		c.CustomAbort(http.StatusNotFound, "Notification not found")
		return
	}
	c.Data["json"] = map[string]interface{}{"success": true, "id": id}
	c.ServeJSON()
}

func (c *Notifications) MarkAllRead() {
	claims, ok := notificationClaims(c)
	if !ok {
		return
	}
	db := connectDB(getConfig(config.Env))
	defer db.Close()

	result, err := db.Exec(`
		UPDATE `+notificationTable()+`
		SET IsRead = 1, ReadAt = COALESCE(ReadAt, GETDATE())
		WHERE UserId = @p1 AND IsRead = 0
	`, claims.UserID)
	if err != nil {
		c.CustomAbort(http.StatusInternalServerError, err.Error())
		return
	}
	updated, _ := result.RowsAffected()
	c.Data["json"] = map[string]interface{}{"success": true, "updated": updated}
	c.ServeJSON()
}

func notificationEntityLink(entityType string, entityID int64) string {
	return fmt.Sprintf("/%s?id=%d", strings.TrimSpace(entityType), entityID)
}

func generateDueDateNotifications() error {
	db := connectDB(getConfig(config.Env))
	defer db.Close()

	_, err := db.Exec(`
		INSERT INTO ` + notificationTable() + `
			(UserId, Type, Title, Message, EntityType, EntityId, Link, IsRead, CreatedAt)
		SELECT
			basket.UserId,
			'due_warning',
			CASE WHEN CAST(item.deathdate AS DATE) < CAST(GETDATE() AS DATE)
				THEN N'Захиалгын хугацаа хэтэрсэн'
				ELSE N'Захиалгын хугацаа ойртлоо' END,
			ISNULL(NULLIF(LTRIM(RTRIM(item.cr4name)), ''), N'Захиалга')
				+ N' · ' + CONVERT(NVARCHAR(10), CAST(item.deathdate AS DATE), 120),
			'BasketItem',
			item.BasketItemId,
			N'/Явц2?itemId=' + CONVERT(NVARCHAR(30), item.BasketItemId),
			0,
			GETDATE()
		FROM ` + getSchema() + `.[BasketItems] item
		INNER JOIN ` + getSchema() + `.[Basket] basket ON basket.BasketId = item.BasketId
		WHERE basket.UserId IS NOT NULL
			AND basket.UserId > 0
			AND ISNULL(item.IsReturned, 0) = 0
			AND ISNULL(item.isArrived, 0) = 0
			AND item.deathdate IS NOT NULL
			AND ISDATE(item.deathdate) = 1
			AND CAST(item.deathdate AS DATE) > '19000101'
			AND CAST(item.deathdate AS DATE) <= DATEADD(DAY, 3, CAST(GETDATE() AS DATE))
			AND NOT EXISTS (
				SELECT 1
				FROM ` + notificationTable() + ` existing
				WHERE existing.UserId = basket.UserId
					AND existing.Type = 'due_warning'
					AND existing.EntityType = 'BasketItem'
					AND existing.EntityId = item.BasketItemId
			)
	`)
	return err
}
