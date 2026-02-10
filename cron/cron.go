package main

import (
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"

	"github.com/calebchiang/bible_server/database"
	"github.com/calebchiang/bible_server/services"
)

type Subscription struct {
	DeviceToken  string
	Timezone     string
	SendHour     int
	LastSentDate *string
}

func main() {
	_ = godotenv.Load()
	log.Println("⏰ Daily Verse cron started")

	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}

	nowUTC := time.Now().UTC()

	rows, err := database.DB.Query(`
		SELECT
			device_token,
			timezone,
			send_hour,
			last_sent_date
		FROM daily_verse_subscriptions;
	`)
	if err != nil {
		log.Fatal("Failed to query subscriptions:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(
			&sub.DeviceToken,
			&sub.Timezone,
			&sub.SendHour,
			&sub.LastSentDate,
		); err != nil {
			log.Println("Scan error:", err)
			continue
		}

		loc, err := time.LoadLocation(sub.Timezone)
		if err != nil {
			log.Println("Invalid timezone:", sub.Timezone)
			continue
		}

		localNow := nowUTC.In(loc)
		// localHour := localNow.Hour()
		localDate := localNow.Format("2006-01-02")

		// // Not this user's hour → skip
		// if localHour != sub.SendHour {
		// 	continue
		// }

		// Already sent today → skip
		if sub.LastSentDate != nil && *sub.LastSentDate == localDate {
			continue
		}

		verse, err := services.FetchRandomVerse(120)
		if err != nil {
			log.Println("Failed to fetch verse:", err)
			continue
		}

		err = services.SendAPNSNotification(
			sub.DeviceToken,
			"Verse of the Day",
			fmt.Sprintf("%s — %s", verse.Text, verse.Reference),
		)

		if err != nil {
			log.Println("❌ APNs send failed:", err)
			continue
		}

		// Mark as sent for today (local date)
		_, err = database.DB.Exec(
			`
			UPDATE daily_verse_subscriptions
			SET last_sent_date = ?, updated_at = ?
			WHERE device_token = ?;
			`,
			localDate,
			nowUTC.Format(time.RFC3339),
			sub.DeviceToken,
		)
		if err != nil {
			log.Println("Failed to update last_sent_date:", err)
		}
	}

	log.Println("✅ Daily Verse cron finished")
}
