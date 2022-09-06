package main

import (
	"fadel-blog-services/controllers/blog"
	"fadel-blog-services/controllers/user"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	var router = gin.Default()
	router.Use(cors.New(cors.Config{
		AllowMethods:    []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Authorization", "Content-type"},
		AllowAllOrigins: true,
	}))

	// User
	router.POST("/login", user.Login)
	router.GET("/users", user.Auth, user.GetUsers)
	router.GET("/users/:id", user.Auth, user.GetUserById)
	router.POST("/users", user.Auth, user.AddUser)
	router.DELETE("/users/:id", user.Auth, user.DeleteUserById)
	router.PATCH("/users/edit/:id", user.Auth, user.EditUserById)
	router.PATCH("/users/updateprofile", user.Auth, user.UpdateProfileImage)

	// Blog
	router.GET("/blog", blog.GetBlogs)
	router.GET("/blog/:slug", blog.GetBlogBySlug)
	router.POST("/blog", user.Auth, blog.AddBlog)
	router.PATCH("/blog/:id", user.Auth, blog.EditBlogById)
	router.DELETE("/blog/:id", user.Auth, blog.DeleteBlogById)
	router.PATCH("/blog/updatethumbnail/:id", user.Auth, blog.UpdateBlogThumbnail)
	router.PATCH("/blog/publish/:id", user.Auth, blog.PublishBlogById)

	router.Run("0.0.0.0:8080")
}
