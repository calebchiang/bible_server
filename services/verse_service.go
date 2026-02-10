package services

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/calebchiang/bible_server/database"
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

type Verse struct {
	Text      string
	Reference string
}

// FetchRandomVerse returns a random tagged verse ≤ maxChars
func FetchRandomVerse(maxChars int) (*Verse, error) {
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
	  AND LENGTH(verses_web.text) <= ?
	ORDER BY RANDOM()
	LIMIT 1;
	`

	row := database.DB.QueryRow(query, tag, maxChars)

	var text, book string
	var chapter, verse int

	if err := row.Scan(&text, &book, &chapter, &verse); err != nil {
		return nil, err
	}

	return &Verse{
		Text:      text,
		Reference: fmt.Sprintf("%s %d:%d", book, chapter, verse),
	}, nil
}
