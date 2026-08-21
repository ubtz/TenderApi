package main

import (
	config "TenderApi/conf"
	_ "TenderApi/routers"

	scheduler "TenderApi/controllers"

	"os"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego/plugins/cors"
)

func main() {
	if err := config.LoadEnvFile(".env"); err != nil {
		beego.Error("Could not load .env:", err)
	}
	beego.SetLevel(beego.LevelDebug)
	beego.BeeLogger.SetLogger("console")
	allowedOrigins := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	allowAllOrigins := allowedOrigins == "*"
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:3001"
	}
	// Log every request
	beego.InsertFilter("*", beego.BeforeRouter, func(ctx *context.Context) {
		beego.Info("Incoming request:", ctx.Input.Method(), ctx.Input.URL())
	}, true)

	beego.InsertFilter("*", beego.BeforeRouter, scheduler.TrackSessionActivity, true)

	// CORS
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		AllowAllOrigins:  allowAllOrigins,
		AllowOrigins:     strings.Split(allowedOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-API-Password"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: !allowAllOrigins,
	}))
	beego.InsertFilter("/get/*", beego.BeforeRouter, scheduler.RequireAuth, true)
	beego.InsertFilter("/post/*", beego.BeforeRouter, scheduler.RequireAuth, true)
	beego.InsertFilter("/put/*", beego.BeforeRouter, scheduler.RequireAuth, true)
	beego.InsertFilter("/delete/*", beego.BeforeRouter, scheduler.RequireAuth, true)
	beego.InsertFilter("/external/*", beego.BeforeRouter, scheduler.RequireAuth, true)
	scheduler.StartDailyJob()
	beego.Run()
}
