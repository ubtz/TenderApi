package routers

import (
	"TenderApi/controllers"

	"github.com/astaxie/beego"
)

func init() {
	// Root route
	beego.Router("/", &controllers.MainController{})

	// Version 1 namespace
	// ns := beego.NewNamespace("/v1",
	// 	beego.NSRouter("/main", &controllers.MainController{}, "GET:FetchDataTable"),
	// 	// beego.NSRouter("/GetnormTypes", &controllers.MainController{}, "GET:FetchnormTypes"),
	// )
	// beego.AddNamespace(ns)

	// Default namespace with proper prefix
	nss := beego.NewNamespace("/get",
		beego.NSRouter("/files", &controllers.GetDocumentsController{}, "GET:GetDocuments"),
		beego.NSRouter("/file", &controllers.DownloadDocumentController{}, "GET:Get"),
		beego.NSRouter("/branches", &controllers.GetBranch{}, "get:GetBranch"),
		beego.NSRouter("/basket", &controllers.GetBasket{}, "get:GetBasket"),
		beego.NSRouter("/GetBasketItems", &controllers.GetBasketItems{}, "get:GetBasketItems"),
		beego.NSRouter("/GetBasketWithTotalPrice", &controllers.GetBasketWithTotalPrice{}, "get:GetBasketWithTotalPrice"),
		beego.NSRouter("/GetAll", &controllers.GetAll{}, "get:GetAll"),
		beego.NSRouter("/Items", &controllers.GetItems{}, "get:GetItems"),
		beego.NSRouter("/GetAllValid", &controllers.GetAllValid{}, "get:GetAllValid"),
		beego.NSRouter("/GetExecTeam", &controllers.GetExecTeam{}, "get:GetExecTeam"),
		beego.NSRouter("/GetTender", &controllers.GetTender{}, "get:GetTender"),
		beego.NSRouter("/GetGeree", &controllers.GetGeree{}, "get:GetGeree"),
		beego.NSRouter("/GetUnelgeeHoroo", &controllers.GetUnelgeeHoroo{}, "get:GetUnelgeeHoroo"),
		beego.NSRouter("/GetBasketItemsById/:basketId", &controllers.GetBasketItemsById{}, "get:GetBasketItemsById"),
		beego.NSRouter("/GetUsers", &controllers.GetUsers{}, "get:GetAllUsers"),
		beego.NSRouter("/Statistic", &controllers.GetStatistic{}, "get:GetStatistic"),
	)

	beego.AddNamespace(nss)

	nsss := beego.NewNamespace("/post",
		beego.NSRouter("/upload", &controllers.UploadFiles{}, "POST:Post"),
		beego.NSRouter("/register", &controllers.Register{}, "POST:PostRegister"),
		beego.NSRouter("/login", &controllers.UserLogin{}, "POST:PostLogin"),
		beego.NSRouter("/addBasket", &controllers.PostBasket{}, "post:PostBasket"),
		beego.NSRouter("/PostBasketItems", &controllers.PostBasketItems{}, "post:PostBasketItems"),
		beego.NSRouter("/PostTender", &controllers.PostTender{}, "post:PostTender"),
		beego.NSRouter("/PostExecTeam", &controllers.PostExecTeam{}, "post:PostExecTeam"),
		beego.NSRouter("/PostGeree", &controllers.PostGeree{}, "post:PostGeree"),
		beego.NSRouter("/PostUnelgeeHoroo", &controllers.PostUnelgeeHoroo{}, "post:PostUnelgeeHoroo"),
		beego.NSRouter("/UserPasswordChange", &controllers.UserPasswordChange{}, "post:Post"),
		beego.NSRouter("/GetUserinfoById", &controllers.GetUserinfoById{}, "post:Post"),
		beego.NSRouter("/UserPasswordRenew", &controllers.UserPasswordRenew{}, "post:Post"),
	)
	beego.AddNamespace(nsss)

	nssss := beego.NewNamespace("/put",
		// Өмнөх маршрутууд
		beego.NSRouter("/basket-item", &controllers.BasketItemController{}, "put:UpdateBasketItemById"),
		beego.NSRouter("/updateBasketValid", &controllers.UpdateBaskedValidation{}, "put:UpdateBaskedValidation"),
		beego.NSRouter("/basketitem/state", &controllers.UpdateBasketItemState{}, "put:Put"), // New route for
		// Шинэ маршрутыг нэмэх
		beego.NSRouter("/UpdateTender/:id", &controllers.UpdateTender{}, "put:Put"),
		beego.NSRouter("/UpdateGeree/:id", &controllers.UpdateGeree{}, "put:Put"),
		beego.NSRouter("/UserInfoUpdate", &controllers.UserInfoUpdate{}, "put:Put"),
		beego.NSRouter("/UpdateBasket", &controllers.UpdateBasket{}, "put:UpdateBasket"),
	)

	// beego.NSRouter("/HB2", &controllers.UpdateХБ_02{}, "PUT:UpdateХБ_02"),
	// beego.NSRouter("/HB3", &controllers.UpdateХБ_03{}, "PUT:UpdateХБ_03"),
	// beego.NSRouter("/HB4", &controllers.UpdateХБ_04{}, "PUT:UpdateХБ_04"),
	// beego.NSRouter("/HB5", &controllers.UpdateХБ_05{}, "PUT:UpdateХБ_05"),
	// beego.NSRouter("/HB6", &controllers.UpdateХБ_06{}, "PUT:UpdateХБ_06"),
	// beego.NSRouter("/HB7", &controllers.UpdateХБ_07{}, "PUT:UpdateХБ_07"),
	// beego.NSRouter("/HB8", &controllers.UpdateХБ_08{}, "PUT:UpdateХБ_08"),
	// beego.NSRouter("/HB9", &controllers.UpdateХБ_09{}, "PUT:UpdateХБ_09"),
	// beego.NSRouter("/HB10", &controllers.UpdateХБ_10{}, "PUT:UpdateХБ_10"),
	// beego.NSRouter("/HB11", &controllers.UpdateХБ_11{}, "PUT:UpdateХБ_11"),
	// beego.NSRouter("/HB12", &controllers.UpdateХБ_12{}, "PUT:UpdateХБ_12"),

	beego.AddNamespace(nssss)

	nsssss := beego.NewNamespace("/delete",
		beego.NSRouter("/file", &controllers.DeleteFiles{}, "delete:Delete"),
		beego.NSRouter("/basket-item", &controllers.DeleteBasketItem{}, "delete:DeleteBasketItem"),
		beego.NSRouter("/deleteBasket", &controllers.DeleteBasket{}, "delete:DeleteBasket"),
	) // Add a valid prefix like "/api"

	beego.AddNamespace(nsssss)

	// beego.InsertFilter("/get/*", beego.BeforeRouter, middlewares.JWTAuth) // Apply JWT Middleware
}
