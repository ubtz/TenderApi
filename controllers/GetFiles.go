package controllers

import (
	config "TenderApi/conf"
	"database/sql"
	"log"
	"time"

	"github.com/astaxie/beego"
)

type GetDocumentsController struct {
	beego.Controller
}

func (c *GetDocumentsController) GetDocuments() {
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	type File struct {
		Id       int64  `json:"Id"`
		FileName string `json:"FileName"`
		FileType string `json:"FileType"`
		FileSize int64  `json:"FileSize"`
	}

	type Document struct {
		DocumentId   int       `json:"DocumentId"`
		UploadedDate time.Time `json:"UploadedDate"`
		GroupName    string    `json:"GroupName"`
		Number       string    `json:"Number"`
		Name         string    `json:"Name"`
		ApprovedBy   string    `json:"ApprovedBy"`
		ApprovedDate string    `json:"ApprovedDate"`
		FollowDate   string    `json:"FollowDate"`
		FileType     string    `json:"FileType"`
		Files        struct {
			Main       []File `json:"Main"`
			Attachment []File `json:"Attachment"`
			Additional []File `json:"Additional"`
		} `json:"Files"`
	}

	// Step 1: Load all documents
	docRows, err := db.Query(`
		SELECT 
			DocumentId,
			UploadedDate,
			GroupName,
			Number,
			Name,
			ApprovedBy,
			ApprovedDate,
			FollowDate,
			FileType
			FROM [Tender].[dbo].[Documents]
			WHERE Visible = 1
			ORDER BY UploadedDate DESC

	`)
	if config.Env == "prod" {
		docRows, err = db.Query(`
		SELECT 
			DocumentId,
			UploadedDate,
			GroupName,
			Number,
			Name,
			ApprovedBy,
			ApprovedDate,
			FollowDate,
			FileType
			FROM [Tender].[logtender].[Documents]
			WHERE Visible = 1
			ORDER BY UploadedDate DESC

	`)
	}
	if err != nil {
		log.Println("Failed to fetch documents:", err)
		c.Data["json"] = map[string]string{"error": "Баримт бичгүүдийг авахад алдаа гарлаа"}
		c.ServeJSON()
		return
	}
	defer docRows.Close()

	docMap := make(map[int]*Document)

	for docRows.Next() {
		var d Document
		var group, number, name, approvedBy, approvedDate, followDate, FileType sql.NullString

		err := docRows.Scan(
			&d.DocumentId, &d.UploadedDate,
			&group, &number, &name,
			&approvedBy, &approvedDate, &followDate, &FileType,
		)
		if err != nil {
			log.Println("Failed to scan document:", err)
			continue
		}

		d.GroupName = nullStringToString(group)
		d.Number = nullStringToString(number)
		d.Name = nullStringToString(name)
		d.ApprovedBy = nullStringToString(approvedBy)
		d.ApprovedDate = nullStringToString(approvedDate)
		d.FollowDate = nullStringToString(followDate)
		d.FileType = nullStringToString(FileType)

		docMap[d.DocumentId] = &d
	}

	// Helper to load files from a table
	loadFiles := func(tableName string, targetKey string) {
		query := `
		SELECT Id, DocumentId, FileName, FileType, DATALENGTH(FileContent) AS FileSize
		FROM [Tender].[dbo].[` + tableName + `]
	`
		if config.Env == "prod" {
			query = `
		SELECT Id, DocumentId, FileName, FileType, DATALENGTH(FileContent) AS FileSize
		FROM [Tender].[logtender].[` + tableName + `]
	`
		}
		rows, err := db.Query(query)
		if err != nil {
			log.Printf("Error loading %s: %v\n", tableName, err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var docId int
			var file File
			var fileName, fileType sql.NullString
			var fileSize sql.NullInt64

			err := rows.Scan(&file.Id, &docId, &fileName, &fileType, &fileSize)
			if err != nil {
				log.Println("Error scanning file row:", err)
				continue
			}

			file.FileName = nullStringToString(fileName)
			file.FileType = nullStringToString(fileType)
			file.FileSize = nullInt64ToInt64(fileSize)

			if doc, ok := docMap[docId]; ok {
				switch targetKey {
				case "Main":
					doc.Files.Main = append(doc.Files.Main, file)
				case "Attachment":
					doc.Files.Attachment = append(doc.Files.Attachment, file)
				case "Additional":
					doc.Files.Additional = append(doc.Files.Additional, file)
				}
			}
		}
	}

	loadFiles("DocUndsen", "Main")
	loadFiles("DocHavsralt", "Attachment")
	loadFiles("DocNemelt", "Additional")

	// Final result
	var result []Document
	for _, d := range docMap {
		result = append(result, *d)
	}

	c.Data["json"] = result
	c.ServeJSON()
}

func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func nullInt64ToInt64(ni sql.NullInt64) int64 {
	if ni.Valid {
		return ni.Int64
	}
	return 0
}
