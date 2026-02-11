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
	LastSentDate *time.Time
}

func main() {
	_ = godotenv.Load()
	log.Println("⏰ Daily Verse cron started")

	// SQLite (Bible content)
	if err := database.Connect(); err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}

	// Postgres (subscriptions)
	if err := database.ConnectPostgres(); err != nil {
		log.Fatal("Failed to connect to Postgres:", err)
	}

	nowUTC := time.Now().UTC()

	rows, err := database.PostgresDB.Query(`
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
		localDate := localNow.Format("2006-01-02")

		// Already sent today → skip
		if sub.LastSentDate != nil && sub.LastSentDate.Format("2006-01-02") == localDate {
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

		// Mark as sent for today
		_, err = database.PostgresDB.Exec(
			`
			UPDATE daily_verse_subscriptions
			SET last_sent_date = $1, updated_at = now()
			WHERE device_token = $2;
			`,
			localNow,
			sub.DeviceToken,
		)
		if err != nil {
			log.Println("Failed to update last_sent_date:", err)
		}
	}

	log.Println("✅ Daily Verse cron finished")
}
