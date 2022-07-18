package main

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	"log"
	"net/http"
	"os"
	"time"
)

type Post struct {
	URL         string
	Title       string
	Timestamp   time.Time
	Body        string
	Description string
}

func ScrapePosts() (posts []Post) {
	res, err := http.Get("https://www.bild.de/themen/personen/franz-josef-wagner/kolumne-17304844.bild.html")
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	doc.Find(".hentry.landscape.t10l").Each(func(i int, s *goquery.Selection) {
		kicker := s.Find(".kicker").Text()
		if kicker != "Post von Wagner" {
			log.Printf("Wrong kicker, skipping: %s", kicker)
			return
		}

		title := s.Find("span.headline").Text()
		url, exists := s.Find("a").Attr("href")
		if !exists {
			log.Printf("could not extract URL")
			return
		}
		post := Post{Title: title, URL: url}
		posts = append(posts, post)
	})

	return posts
}

func ScrapeEntry(post *Post) {
	res, err := http.Get(fmt.Sprintf("https://www.bild.de/%s", post.URL))
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Fatalf("status code error: %d %s", res.StatusCode, res.Status)
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	doc.Find("figure").Remove()
	doc.Find("aside").Remove()
	doc.Find("a").Remove()

	article := doc.Find("article")

	timestamp, ok := article.Find("time.datetime").Attr("datetime")
	if !ok {
		log.Printf("could not extract timestamp")
	}
	datetime, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		log.Fatal(err)
	}
	post.Timestamp = datetime

	body, err := article.Find("div.article-body").Html()
	if err != nil {
		log.Fatal(err)
	}
	post.Body = body

	post.Description = article.Find("div.article-body").Children().First().Text()
}

func main() {
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Post von Wagner",
		Link:        &feeds.Link{Href: "https://www.bild.de/themen/personen/franz-josef-wagner/kolumne-17304844.bild.html"},
		Description: "Franz Josef Wagner ist seit 2001 Chefkolumnist im Hause Axel Springer.",
		Author:      &feeds.Author{Name: "Franz Josef Wagner", Email: "fjwagner@bild.de"},
		Created:     now,
	}

	posts := ScrapePosts()
	for i, post := range posts {
		ScrapeEntry(&post)
		item := feeds.Item{
			Title:       post.Title,
			Link:        &feeds.Link{Href: fmt.Sprintf("https://www.bild.de/%s", post.URL)},
			Author:      &feeds.Author{Name: "Franz Josef Wagner", Email: "fjwagner@bild.de"},
			Description: post.Description,
			Id:          post.URL,
			Created:     post.Timestamp,
			Content:     post.Body,
		}
		feed.Items = append(feed.Items, &item)
		fmt.Printf("Post %d: %s (%s)\n", i, post.Title, post.URL)
	}

	rss, err := feed.ToRss()
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Create("fjw.rss")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(rss)
	if err != nil {
		log.Fatal(err)
	}
}
