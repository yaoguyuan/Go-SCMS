package main

import (
	"auth/controllers"
	"auth/initializers"
	"auth/middlewares"

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
}

func main() {
	r := gin.Default()

	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/signup", controllers.SignUp) // Log Audit
		apiGroup.POST("/login", controllers.Login)   // Log Audit
		apiGroup.POST("/send_code", controllers.SendCode)
		apiGroup.POST("/verify_code", controllers.VerifyCode)
	}

	userInterfaceGroup := apiGroup.Group("/ui", middlewares.RequireAuthentication, middlewares.RequireAuthorization)
	{
		userInterfaceGroup.GET("/myself", controllers.Fetch)
		userInterfaceGroup.PUT("/myself", controllers.Modify) // Log Audit
		userInterfaceGroup.GET("/users", controllers.FetchUsers)
		userInterfaceGroup.GET("/users/:id", controllers.FetchUser)
		userInterfaceGroup.GET("/avatar/:id", controllers.GetAvatar)
		userInterfaceGroup.POST("/avatar", controllers.UploadAvatar)
		userInterfaceGroup.GET("/articles", controllers.FetchArticles)
		userInterfaceGroup.GET("/articles/:id", controllers.FetchUserArticles)
		userInterfaceGroup.POST("/articles", controllers.PostArticle)     // Log Audit
		userInterfaceGroup.DELETE("/articles", controllers.RemoveArticle) // Log Audit
		userInterfaceGroup.POST("/articles/like", controllers.LikeArticle)
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
