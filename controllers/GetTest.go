package controllers

import (
	"log"

	"github.com/astaxie/beego"
)

type GetTestController struct {
	beego.Controller
}

// GET /get/test
func (c *GetTestController) Get() {
	log.Println("✅ GetTest endpoint HIT")

	// ✅ Query params (WORKS in all Beego v1)
	log.Println("📌 Query Params:")
	query := c.Ctx.Request.URL.Query()
	for k, v := range query {
		log.Printf("  %s = %v\n", k, v)
	}

	// ✅ Headers
	log.Println("📌 Headers:")
	for k, v := range c.Ctx.Request.Header {
		log.Printf("  %s = %v\n", k, v)
	}

	// ✅ Body (usually empty for GET)
	body := c.Ctx.Input.RequestBody
	log.Printf("📌 Body: %s\n", string(body))

	// ✅ Client info
	log.Println("📌 Client IP:", c.Ctx.Input.IP())
	log.Println("📌 Method:", c.Ctx.Input.Method())

	// Response
	c.Ctx.Output.Body([]byte("GetTest received. Check server logs."))
}
