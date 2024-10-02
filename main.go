package main

import (
	"encoding/csv"
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
	"strings"
	"time"
)

type Post struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Timestamp   time.Time `json:"timestamp"`
	Body        string    `json:"-"`
	Description string    `json:"-"`
}

func ScrapePosts() (posts []Post) {
	res, err := http.Get("https://www.bild.de/autor/franz-josef-wagner")
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

	doc.Find("article .author-recommendation__article").Each(func(i int, s *goquery.Selection) {
		kicker := s.Find(".teaser__title__kicker").Text()
		if kicker != "Post von Wagner" {
			log.Printf("Wrong kicker, skipping: %s\n", kicker)
			return
		}

		title := s.Find(".teaser__title__headline").Text()
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

func ScrapeArchive() (posts []Post) {
	csvFile, err := os.Create("fjw.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer csvFile.Close()

	csvWriter := csv.NewWriter(csvFile)
	defer csvWriter.Flush()

	startDate := time.Date(2006, 1, 1, 0, 0, 0, 0, &time.Location{})
	//startDate := time.Date(2008, 2, 1, 0, 0, 0, 0, &time.Location{})

	//endDate := time.Date(2008, 2, 15, 0, 0, 0, 0, &time.Location{})
	endDate := time.Now()

	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		fmt.Printf("Working on: %s\n", fmt.Sprintf("https://www.bild.de/archive/%d/%d/%d/index.html", date.Year(), date.Month(), date.Day()))

		res, err := http.Get(fmt.Sprintf("https://www.bild.de/archive/%d/%d/%d/index.html", date.Year(), date.Month(), date.Day()))
		if err != nil {
			log.Print(err)
			continue
		}
		defer res.Body.Close()

		if res.StatusCode != 200 {
			log.Printf("status code error: %d %s", res.StatusCode, res.Status)
			continue
		}

		doc, err := goquery.NewDocumentFromReader(res.Body)
		if err != nil {
			log.Print(err)
			continue
		}

		doc.Find(".txt a").Each(func(i int, s *goquery.Selection) {
			if strings.HasPrefix(s.Text(), "Post von Wagner") {
				url, exists := s.Attr("href")
				if !exists {
					log.Printf("could not extract URL")
					return
				}
				post := Post{Title: strings.TrimPrefix(s.Text(), "Post von Wagner: "), URL: url}
				posts = append(posts, post)
				fmt.Printf("New post: %s (%s)", post.Title, post.URL)

				csvWriter.Write([]string{post.URL, post.Title})
			}
		})

	}

	return posts
}

func ScrapeEntry(post *Post) {
	res, err := http.Get(post.URL)
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
		timestamp, ok = article.Find("time.authors__pubdate").Attr("datetime")
	}
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
	if len(post.Description) == 0 {
		article.Find("div.txt").Children().Filter("p:nth-last-child(-n+2)").Remove()
		post.Description = article.Find("div.txt").Children().Filter("p").Text()
	}
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
		_, err := twitter.PostTweet(fmt.Sprintf("%s %s", post.Title, post.URL), url.Values{})
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

	//posts := ScrapeArchive()
	posts := ScrapePosts()

	for i, _ := range posts {
		ScrapeEntry(&posts[i])
		item := feeds.Item{
			Title:       posts[i].Title,
			Link:        &feeds.Link{Href: posts[i].URL},
			Author:      &feeds.Author{Name: "Franz Josef Wagner", Email: "fjwagner@bild.de"},
			Description: posts[i].Description,
			Id:          posts[i].URL,
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
}
