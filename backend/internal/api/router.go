package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Content-Type"},
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	bookHandler := NewBookHandler()
	books := r.Group("/books")
	{
		books.GET("", bookHandler.List)
		books.GET("/:id", bookHandler.Get)
		books.POST("", bookHandler.Create)
		books.PUT("/:id", bookHandler.Update)
		books.DELETE("/:id", bookHandler.Delete)
	}

	return r
}
