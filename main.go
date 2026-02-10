package main

import (
	"log"

	"github.com/calebchiang/bible_server/database"
	"github.com/calebchiang/bible_server/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	routes.VerseRoutes(r)

	log.Println("Bible server running on :8080")
	r.Run(":8080")
}
