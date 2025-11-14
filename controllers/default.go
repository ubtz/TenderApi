package controllers

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/astaxie/beego"
	mssql "github.com/denisenkom/go-mssqldb"
)

type MainController struct {
	beego.Controller
}
type DBConfig struct {
	Server   string
	Port     int
	User     string
	Password string
	Database string
}

func getConfig(env string) DBConfig {
	if env == "prod" {
		return DBConfig{
			Server:   "192.168.4.123",
			Port:     1433,
			User:     "logtender",
			Password: "UBjsc@tender.nrp",
			Database: "tender",
		}
	}
	// Default to test
	return DBConfig{
		Server:   "172.30.30.30",
		Port:     1433,
		User:     "sa",
		Password: "test",
		Database: "test",
	}
}

func connectDB(cfg DBConfig) *sql.DB {
	connString := fmt.Sprintf(
		"server=%s;port=%d;user id=%s;password=%s;database=%s;encrypt=disable",
		cfg.Server, cfg.Port, cfg.User, cfg.Password, cfg.Database,
	)

	connector, err := mssql.NewConnector(connString)
	if err != nil {
		log.Fatal("Connector creation failed:", err)
	}

	db := sql.OpenDB(connector)

	if err := db.Ping(); err != nil {
		log.Fatal("Database connection failed:", err)
	}

	fmt.Println("Database connection successful!")
	return db
}
func (c *MainController) FetchDataTable() {

	env := "test" // change to "prod" for production
	cfg := getConfig(env)
	db := connectDB(cfg)

	// defer db.Close() if needed for cleanup later
	_ = db // use the db connection as needed

	//UPDATE
	// 	updateQuery := `
	// 	UPDATE UserLogin
	// 	SET password = @p1
	// 	WHERE username = @p2
	// `

	// 	// Use the same data that was recently posted (example purposes)
	// 	username := "JohnDoe"           // Replace with the recently posted username
	// 	newPassword := "newPassword123" // Replace with the new password

	// 	// Execute the UPDATE query
	// 	result, err := db.Exec(updateQuery, newPassword, username)
	// 	if err != nil {
	// 		log.Fatal("Failed to execute update query:", err)
	// 	}

	// 	// Get the number of rows affected
	// 	rowsAffected, err := result.RowsAffected()
	// 	if err != nil {
	// 		log.Fatal("Failed to fetch rows affected:", err)
	// 	}

	// 	fmt.Printf("Updated %d row(s) in UserLogin successfully.\n", rowsAffected)

	// 	// Close the database connection when done
	// 	defer db.Close()
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//DELETE
	// deleteQuery := `
	// 	DELETE FROM UserLogin
	// 	WHERE username = @p1 AND password = @p2
	// `

	// // Use the same data that was recently posted (for example purposes)
	// username := "JohnDoe"          // Replace with the recently posted username
	// passwordValue := "password123" // Replace with the recently posted password

	// // Execute the DELETE query
	// result, err := db.Exec(deleteQuery, username, passwordValue)
	// if err != nil {
	// 	log.Fatal("Failed to execute delete query:", err)
	// }

	// // Get the number of rows affected
	// rowsAffected, err := result.RowsAffected()
	// if err != nil {
	// 	log.Fatal("Failed to fetch rows affected:", err)
	// }

	// fmt.Printf("Deleted %d row(s) from UserLogin successfully.\n", rowsAffected)
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//POST
	// insertQuery := `
	// 	INSERT INTO UserLogin (Username, Password)
	// 	VALUES (@p1, @p2)
	// `

	// // Example user details
	// username := "JohnDoe"
	// passwordValue := "password123"

	// // Execute the INSERT query
	// result, err := db.Exec(insertQuery, username, passwordValue)
	// if err != nil {
	// 	log.Fatal("Failed to execute insert query:", err)
	// }

	// // Get the number of rows affected
	// rowsAffected, err := result.RowsAffected()
	// if err != nil {
	// 	log.Fatal("Failed to fetch rows affected:", err)
	// }

	// fmt.Printf("Inserted %d row(s) into UserLogin successfully.\n", rowsAffected)

	// // Close the database connection when done
	// defer db.Close()
	////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
	//GET
	query := "SELECT [PersonID], [LastName], [FirstName], [Address], [City] FROM Persons;"

	// Execute the query and get a result set
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Error executing query:", err)
	}
	defer rows.Close() // Ensure rows are closed after processing

	// Define a struct to hold query results
	type Person struct {
		PersonID  int    `json:"person_id"`
		LastName  string `json:"last_name"`
		FirstName string `json:"first-name"`
		Address   string `json:"address"`
		City      string `json:"city"`
	}

	var result []Person
	for rows.Next() {
		var person Person
		if err := rows.Scan(&person.PersonID, &person.LastName, &person.FirstName, &person.Address, &person.City); err != nil {
			log.Fatal("Error scanning row:", err)
		}
		result = append(result, person)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		log.Fatal("Error during row iteration:", err)
	}

	// Send JSON response as an array
	c.Data["json"] = result
	c.ServeJSON()

	// dbcon = sql.OpenDB(connector)
	//   rows, err:= {
	// 	fmt.PrintLn("aldaa")
	//   }
}

func (c *MainController) FetchnormTypes() {

	server := "172.30.30.30" // SQL Server instance name
	port := 1433             // SQL Server default port
	user := "sa"             // Your MSSQL username
	password := "test"       // Your MSSQL password
	database := "test"

	// Convert port to string
	// connString := fmt.Sprintf("server=%s;user id=%s;password=%s;port=%s;database=%s;encrypt=disable;sslmode=disable;connection+timeout=60;log=63", server, user, password, port, database)
	connString := fmt.Sprintf("server=%s;port=%d;user id=%s;password=%s;database=%s;encrypt=disable", server, port, user, password, database)

	// fmt.Println(connString)

	connector, err := mssql.NewConnector(connString)
	if nil != err {
		log.Fatal("error", err.Error())
	}

	// Open a new database connection
	db := sql.OpenDB(connector)

	// Ping the database to check if the connection is successful
	err = db.Ping()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	} else {
		fmt.Println("Database connection successful!")
	}

	query := "SELECT [PersonID], [LastName], [FirstName], [Address], [City] FROM Persons;"

	// Execute the query and get a result set
	rows, err := db.Query(query)
	if err != nil {
		log.Fatal("Error executing query:", err)
	}
	defer rows.Close() // Ensure rows are closed after processing

	// Define a struct to hold query results
	type Person struct {
		PersonID  int    `json:"person_id"`
		LastName  string `json:"last_name"`
		FirstName string `json:"first-name"`
		Address   string `json:"address"`
		City      string `json:"city"`
	}

	var result []Person
	for rows.Next() {
		var person Person
		if err := rows.Scan(&person.PersonID, &person.LastName, &person.FirstName, &person.Address, &person.City); err != nil {
			log.Fatal("Error scanning row:", err)
		}
		result = append(result, person)
	}

	// Check for errors during iteration
	if err := rows.Err(); err != nil {
		log.Fatal("Error during row iteration:", err)
	}

	// Send JSON response as an array
	c.Data["json"] = result
	c.ServeJSON()

}
