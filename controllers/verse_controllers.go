package controllers

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/calebchiang/bible_server/database"
	"github.com/gin-gonic/gin"
)

var tags = []string{
	"anxiety",
	"encouragement",
	"forgiveness",
	"healing",
	"hope",
	"peace",
	"stress",
}

type VerseResponse struct {
	Text      string `json:"text"`
	Reference string `json:"reference"`
}

func GetRandomVerse(c *gin.Context) {
	rand.Seed(time.Now().UnixNano())
	tag := tags[rand.Intn(len(tags))]

	query := `
	SELECT
		verses_web.text,
		books.name,
		chapters.number,
		verses_web.verse_number
	FROM verses_web
	JOIN chapters ON verses_web.chapter_id = chapters.id
	JOIN books ON chapters.book_id = books.id
	JOIN verse_tags ON verses_web.id = verse_tags.verse_id
	JOIN tags ON verse_tags.tag_id = tags.id
	WHERE tags.name = ?
	ORDER BY RANDOM()
	LIMIT 1;
	`

	row := database.DB.QueryRow(query, tag)

	var text string
	var book string
	var chapter int
	var verse int

	err := row.Scan(&text, &book, &chapter, &verse)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch verse",
		})
		return
	}

	c.JSON(http.StatusOK, VerseResponse{
		Text:      text,
		Reference: book + " " + fmt.Sprintf("%d:%d", chapter, verse),
	})
}

type SubscribeRequest struct {
	DeviceToken string `json:"device_token"`
	Timezone    string `json:"timezone"`
	SendHour    int    `json:"send_hour"`
}

func SubscribeToDailyVerse(c *gin.Context) {
	var req SubscribeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	// Validate required fields
	if req.DeviceToken == "" || req.Timezone == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "device_token and timezone are required",
		})
		return
	}

	if req.SendHour < 0 || req.SendHour > 23 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "send_hour must be between 0 and 23",
		})
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)

	query := `
	INSERT INTO daily_verse_subscriptions (
		device_token,
		timezone,
		send_hour,
		last_sent_date,
		created_at,
		updated_at
	) VALUES (?, ?, ?, NULL, ?, ?)
	ON CONFLICT(device_token) DO UPDATE SET
		timezone = excluded.timezone,
		send_hour = excluded.send_hour,
		last_sent_date = NULL,
		updated_at = excluded.updated_at;
	`

	_, err := database.DB.Exec(
		query,
		req.DeviceToken,
		req.Timezone,
		req.SendHour,
		now,
		now,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to save subscription",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

type SubscriptionRow struct {
	DeviceToken  string  `json:"device_token"`
	Timezone     string  `json:"timezone"`
	SendHour     int     `json:"send_hour"`
	LastSentDate *string `json:"last_sent_date"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

func GetAllSubscriptions(c *gin.Context) {
	query := `
	SELECT
		device_token,
		timezone,
		send_hour,
		last_sent_date,
		created_at,
		updated_at
	FROM daily_verse_subscriptions
	ORDER BY created_at DESC;
	`

	rows, err := database.DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to query subscriptions",
		})
		return
	}
	defer rows.Close()

	subscriptions := []SubscriptionRow{}

	for rows.Next() {
		var row SubscriptionRow
		if err := rows.Scan(
			&row.DeviceToken,
			&row.Timezone,
			&row.SendHour,
			&row.LastSentDate,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to scan subscription row",
			})
			return
		}

		subscriptions = append(subscriptions, row)
	}

	c.JSON(http.StatusOK, subscriptions)
}

func DeleteAllSubscriptions(c *gin.Context) {
	result, err := database.DB.Exec(
		"DELETE FROM daily_verse_subscriptions;",
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete subscriptions",
		})
		return
	}

	rowsAffected, _ := result.RowsAffected()

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"rows_deleted": rowsAffected,
	})
}
