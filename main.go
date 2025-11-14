package main

import (
	_ "TenderApi/routers"

	config "TenderApi/conf"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/plugins/cors"
)

func main() {
	beego.SetLevel(beego.LevelDebug)
	beego.BeeLogger.SetLogger("console")
	config.Env = "test" // Set the environment variable  here
	// Log every request
	beego.InsertFilter("*", beego.BeforeRouter, func(ctx *context.Context) {
		beego.Info("Incoming request:", ctx.Input.Method(), ctx.Input.URL())
	}, true)

	// CORS
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowOrigins:     []string{"*"}, // or "*"
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	beego.Run()
}
