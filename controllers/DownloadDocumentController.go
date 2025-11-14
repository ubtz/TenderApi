package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/astaxie/beego"
)

type DownloadDocumentController struct {
	beego.Controller
}

func (c *DownloadDocumentController) Get() {
	idStr := c.GetString("id")
	fileType := c.GetString("type") // main, attachment, additional

	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.CustomAbort(400, "Invalid ID")
		return
	}

	tableName, err := resolveTableName(fileType)
	if err != nil {
		c.CustomAbort(400, err.Error())
		return
	}

	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Safe query using fmt only for validated tableName
	query := fmt.Sprintf(`
		SELECT FileName, FileType, FileContent
		FROM [Tender].[dbo].[%s]
		WHERE Id = @p1
	`, tableName)
	if config.Env == "prod" {
		query = fmt.Sprintf(`
		SELECT FileName, FileType, FileContent
		FROM [Tender].[logtender].[%s]
		WHERE Id = @p1
	`, tableName)
	}
	var fileName, contentType string
	var fileContent []byte

	err = db.QueryRow(query, id).Scan(&fileName, &contentType, &fileContent)
	if err != nil {
		if err == sql.ErrNoRows {
			c.CustomAbort(404, "File not found")
		} else {
			c.CustomAbort(500, "Error fetching file: "+err.Error())
		}
		return
	}

	// Set headers safely
	c.Ctx.Output.Header("Content-Type", contentType)

	// Safely encode filename for header
	escapedFileName := url.PathEscape(fileName)
	disposition := fmt.Sprintf("inline; filename*=UTF-8''%s", escapedFileName)
	c.Ctx.Output.Header("Content-Disposition", disposition)

	c.Ctx.Output.Body(fileContent)
}

// resolveTableName ensures only valid table names are used
func resolveTableName(fileType string) (string, error) {
	switch strings.ToLower(fileType) {
	case "main":
		return "DocUndsen", nil
	case "attachment":
		return "DocHavsralt", nil
	case "additional":
		return "DocNemelt", nil
	default:
		return "", errors.New("invalid file type. Must be 'main', 'attachment', or 'additional'")
	}
}
