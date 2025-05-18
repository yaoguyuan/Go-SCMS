package main

import (
	"auth/controllers"
	"auth/initializers"
	"auth/middlewares"
	"auth/tasks"

	"github.com/gin-gonic/gin"
)

func init() {
	initializers.LoadEnvVar()
	initializers.InitLogger()
	initializers.InitEmail()
	initializers.ConnectToDB()
	initializers.ConnectToRedis()
	initializers.SyncDB()
	initializers.InitCasbin()
	initializers.EnsureAvatarDefault()
	tasks.InitSeckillProcessor()
}

func main() {
	r := gin.Default()
	// r.Use(middlewares.RequestID) // Generate a unique request ID for each request

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/signup", controllers.SignUp) // Log Audit
		apiGroup.POST("/login", controllers.Login)   // Log Audit
		// *** Using Redis for Session Management ***
		apiGroup.POST("/send_code", controllers.SendCode)
		apiGroup.POST("/verify_code", controllers.VerifyCode)
		// ******************************************
	}

	userInterfaceGroup := apiGroup.Group("/ui", middlewares.RequireAuthentication, middlewares.RequireAuthorization)
	{
		userInterfaceGroup.GET("/myself", controllers.Fetch)
		userInterfaceGroup.PUT("/myself", controllers.Modify) // Log Audit
		// ************** Using Redis for Sign in **************
		userInterfaceGroup.POST("/signin", controllers.SignIn)
		userInterfaceGroup.GET("/signin/count", controllers.SignInCount)
		userInterfaceGroup.POST("/signin/award", controllers.SignInAward)
		// *****************************************************
		userInterfaceGroup.GET("/users", controllers.FetchUsers)
		// ************** Using Redis for Caching **************
		userInterfaceGroup.GET("/users/:id", controllers.FetchUser)
		// *****************************************************
		userInterfaceGroup.GET("/avatar/:id", controllers.GetAvatar)
		userInterfaceGroup.POST("/avatar", controllers.UploadAvatar)
		userInterfaceGroup.POST("/subscribe", controllers.Subscribe)
		// ************** Using Redis for Seckill **************
		userInterfaceGroup.POST("/seckill", controllers.Seckill)
		// *****************************************************
		userInterfaceGroup.GET("/discounts/:id", controllers.FetchUserDiscounts)
		userInterfaceGroup.POST("/discounts", controllers.PostDiscount)
		userInterfaceGroup.GET("/articles/:id", controllers.FetchUserArticles)
		userInterfaceGroup.POST("/articles", controllers.PostArticle)     // Log Audit
		userInterfaceGroup.DELETE("/articles", controllers.RemoveArticle) // Log Audit
		// ************** Using Redis for Leaderboard **************
		userInterfaceGroup.POST("/articles/like", controllers.LikeArticle)
		// *********************************************************
		userInterfaceGroup.POST("/articles/comment", controllers.PostComment)     // Log Audit
		userInterfaceGroup.DELETE("/articles/comment", controllers.RemoveComment) // Log Audit
	}

	backgroundGroup := apiGroup.Group("/bg", middlewares.RequireAuthentication, middlewares.RequireAuthorization)
	{
		backgroundGroup.GET("/users", controllers.GetUsers)
		backgroundGroup.DELETE("/users", controllers.DelUser) // Log Audit
		backgroundGroup.GET("/rules", controllers.GetDenials)
		backgroundGroup.POST("/rules", controllers.AddDenial)   // Log Audit
		backgroundGroup.DELETE("/rules", controllers.DelDenial) // Log Audit
		backgroundGroup.GET("/articles", controllers.GetArticles)
		backgroundGroup.PUT("/articles", controllers.SetArticleStatus) // Log Audit
		backgroundGroup.GET("/comments", controllers.GetComments)
		backgroundGroup.PUT("/comments", controllers.SetCommentStatus) // Log Audit
		backgroundGroup.GET("/logs", controllers.DownloadLogFile)
	}

	r.Run(":8080")
}
