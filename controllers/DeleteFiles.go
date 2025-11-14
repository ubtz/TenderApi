package controllers

import (
	config "TenderApi/conf"
	"strconv"

	"github.com/astaxie/beego"
)

type DeleteFiles struct {
	beego.Controller
}

func (c *DeleteFiles) Delete() {
	// Get the "id" parameter from query
	idStr := c.GetString("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.CustomAbort(400, "Invalid ID")
		return
	}

	// Connect to database
	cfg := getConfig(config.Env)
	db := connectDB(cfg)
	defer db.Close()

	// Perform the update
	query := `UPDATE [Tender].[dbo].[Documents] SET Visible = 0 WHERE DocumentId = @p1`

	if config.Env == "prod" {
		query = `UPDATE [Tender].[logtender].[Documents] SET Visible = 0 WHERE DocumentId = @p1`
	}

	result, err := db.Exec(query, id)
	if err != nil {
		c.CustomAbort(500, "Failed to delete file: "+err.Error())
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.CustomAbort(404, "No document found with the given ID")
		return
	}

	c.Data["json"] = map[string]interface{}{
		"success": true,
		"message": "File marked as invisible (deleted) successfully",
	}
	c.ServeJSON()
}
