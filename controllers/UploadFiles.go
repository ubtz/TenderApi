package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"io/ioutil"
	"log"
	"mime/multipart"
	"time"

	"github.com/astaxie/beego"
	_ "github.com/denisenkom/go-mssqldb"
)

type UploadFiles struct {
	beego.Controller
}

func (c *UploadFiles) Post() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Parse multipart form
	err := c.Ctx.Request.ParseMultipartForm(32 << 20) // 32 MB
	if err != nil {
		log.Println("ParseMultipartForm error:", err)
		c.Data["json"] = map[string]string{"error": "Формат задлахад алдаа гарлаа"}
		c.ServeJSON()
		return
	}

	// Extract form fields
	group := c.GetString("group")
	number := c.GetString("number")
	name := c.GetString("name")
	fileType := c.GetString("type")
	approvedBy := c.GetString("approvedBy")
	approvedDate := c.GetString("approvedDate")
	followDate := c.GetString("followDate")
	now := time.Now()

	// Insert metadata into Documents and get new DocumentId
	var documentId int
	docQuery := `
	INSERT INTO [Tender].[dbo].[Documents] (
		UploadedDate, GroupName, Number, Name,
		ApprovedBy, ApprovedDate, FollowDate, FileType, Visible
	)
	OUTPUT INSERTED.DocumentId
	VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9);
`
	if config.Env == "prod" {
		docQuery = `
	INSERT INTO [Tender].[logtender].[Documents] (
		UploadedDate, GroupName, Number, Name,
		ApprovedBy, ApprovedDate, FollowDate, FileType, Visible
	)
	OUTPUT INSERTED.DocumentId
	VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9);
`
	}
	err = db.QueryRow(
		docQuery,
		sql.Named("p1", now),
		sql.Named("p2", group),
		sql.Named("p3", number),
		sql.Named("p4", name),
		sql.Named("p5", approvedBy),
		sql.Named("p6", approvedDate),
		sql.Named("p7", followDate),
		sql.Named("p8", fileType),
		sql.Named("p9", 1), // insert int 1
	).Scan(&documentId)

	if err != nil {
		log.Println("Failed to insert document metadata:", err)
		c.Data["json"] = map[string]string{"error": "Мэдээлэл хадгалахад алдаа гарлаа"}
		c.ServeJSON()
		return
	}

	// Save files into corresponding tables
	saveFiles := func(fieldName, tableName string) error {
		files := c.Ctx.Request.MultipartForm.File[fieldName]
		for _, fHeader := range files {
			err := insertFile(db, fHeader, tableName, now, documentId)
			if err != nil {
				log.Printf("Failed to save file to %s: %v\n", tableName, err)
				return err
			}
		}
		return nil
	}

	if err := saveFiles("main[]", "DocUndsen"); err != nil {
		c.Data["json"] = map[string]string{"error": "Үндсэн файл хадгалахад алдаа гарлаа"}
		c.ServeJSON()
		return
	}
	if err := saveFiles("attachment[]", "DocHavsralt"); err != nil {
		c.Data["json"] = map[string]string{"error": "Хавсралт файл хадгалахад алдаа гарлаа"}
		c.ServeJSON()
		return
	}
	if err := saveFiles("additional[]", "DocNemelt"); err != nil {
		c.Data["json"] = map[string]string{"error": "Нэмэлт файл хадгалахад алдаа гарлаа"}
		c.ServeJSON()
		return
	}

	c.Data["json"] = map[string]string{"message": "Бүх файл амжилттай хадгалагдлаа"}
	c.ServeJSON()
}

// Helper function to insert file into given table
func insertFile(db *sql.DB, fileHeader *multipart.FileHeader, tableName string, uploadedDate time.Time, documentId int) error {
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO [Tender].[dbo].[` + tableName + `] (
			FileName, FileType, FileContent, UploadedDate, DocumentId
		)
		VALUES (@p1, @p2, @p3, @p4, @p5);
	`
	if config.Env == "prod" {
		query = `
		INSERT INTO [Tender].[logtender].[` + tableName + `] (
			FileName, FileType, FileContent, UploadedDate, DocumentId
		)
		VALUES (@p1, @p2, @p3, @p4, @p5);
	`
	}
	_, err = db.Exec(
		query,
		sql.Named("p1", fileHeader.Filename),
		sql.Named("p2", fileHeader.Header.Get("Content-Type")),
		sql.Named("p3", fileBytes),
		sql.Named("p4", uploadedDate),
		sql.Named("p5", documentId),
	)

	return err
}
