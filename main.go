package main

import (
	"encoding/json"
	"fmt"
	"github.com/ChimeraCoder/anaconda"
	"github.com/PuerkitoBio/goquery"
	"github.com/gorilla/feeds"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Post struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Timestamp   time.Time `json:"timestamp"`
	Body        string    `json:"-"`
	Description string    `json:"-"`
}

func (p *Post) CompleteURL() string {
	return fmt.Sprintf("https://www.bild.de%s", p.URL)
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
	res, err := http.Get(post.CompleteURL())
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

	body := article.Find("div.article-body")
	if body.Children().Last().Text() != "Ihr Franz Josef Wagner" {
		body.Children().Last().Remove()
	}

	post.Body, err = body.Html()
	if err != nil {
		log.Fatal(err)
	}

	post.Description = article.Find("div.article-body").Children().First().Text()
}

func tweet(post Post) {
	twitter := anaconda.NewTwitterApiWithCredentials(
		os.Getenv("OAUTH_TOKEN"),
		os.Getenv("OAUTH_TOKEN_SECRET"),
		os.Getenv("APP_KEY"),
		os.Getenv("APP_SECRET"))
	_, err := twitter.GetSelf(url.Values{})
	if err != nil {
		log.Print(err)
		return
	}

	var lastPost Post

	file, err := ioutil.ReadFile("tweet.json")
	if err != nil {
		log.Print(err)
	}
	err = json.Unmarshal(file, &lastPost)
	if err != nil {
		log.Print(err)
	}

	if post.Timestamp.After(lastPost.Timestamp) && post.URL != lastPost.URL {
		_, err := twitter.PostTweet(fmt.Sprintf("%s %s", post.Title, post.CompleteURL()), url.Values{})
		if err != nil {
			log.Print(err)
			return
		}
		log.Printf("New tweet was created\n")
	}

	lastPost = post
	file, err = json.Marshal(lastPost)
	if err != nil {
		log.Print(err)
		return
	}

	err = ioutil.WriteFile("tweet.json", file, 0644)
	if err != nil {
		log.Print(err)
		return
	}
}

func main() {
	feed := &feeds.Feed{
		Title:       "Post von Wagner",
		Link:        &feeds.Link{Href: "https://www.bild.de/themen/personen/franz-josef-wagner/kolumne-17304844.bild.html"},
		Description: "Franz Josef Wagner ist seit 2001 Chefkolumnist im Hause Axel Springer",
		Author:      &feeds.Author{Name: "Franz Josef Wagner", Email: "fjwagner@bild.de"},
	}

	posts := ScrapePosts()
	for i, _ := range posts {
		ScrapeEntry(&posts[i])
		item := feeds.Item{
			Title:       posts[i].Title,
			Link:        &feeds.Link{Href: posts[i].CompleteURL()},
			Author:      &feeds.Author{Name: "Franz Josef Wagner", Email: "fjwagner@bild.de"},
			Description: posts[i].Description,
			Id:          posts[i].CompleteURL(),
			Created:     posts[i].Timestamp,
			Content:     posts[i].Body,
		}
		feed.Add(&item)
		if item.Created.After(feed.Created) {
			feed.Created = item.Created
		}
		fmt.Printf("Post %d: %s (%s)\n", i, posts[i].Title, posts[i].URL)
	}

	file, err := os.Create("fjw.rss")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	err = feed.WriteRss(file)
	if err != nil {
		log.Fatal(err)
	}

	if len(posts) > 0 {
		tweet(posts[0])
	}
}
