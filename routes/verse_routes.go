package routes

import (
	"github.com/calebchiang/bible_server/controllers"
	"github.com/gin-gonic/gin"
)

func VerseRoutes(r *gin.Engine) {
	verse := r.Group("/verse")
	{
		verse.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "Verse route working"})
		})

		verse.GET("/random", controllers.GetRandomVerse)
		verse.POST("/subscribe", controllers.SubscribeToDailyVerse)
		verse.GET("/subscriptions", controllers.GetAllSubscriptions)
	}
}
