package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var db *sql.DB

type OLXAd struct {
	Title    string `json:"title"`
	Image    string `json:"image"`
	Price    int    `json:"price"`
	Location string `json:"location"`
}

func randomPage() ([]OLXAd, error) {
	var ads []OLXAd

	urls := []string{
		"https://www.olx.com.br/eletronicos-e-celulares",
		"https://www.olx.com.br/para-a-sua-casa",
		"https://www.olx.com.br/eletro",
		"https://www.olx.com.br/moveis",
		"https://www.olx.com.br/esportes-e-lazer",
		"https://www.olx.com.br/musica-e-hobbies",
		"https://www.olx.com.br/agro-e-industria",
		"https://www.olx.com.br/moda-e-beleza",
		"https://www.olx.com.br/artigos-infantis",
		"https://www.olx.com.br/animais-de-estimacao",
		"https://www.olx.com.br/cameras-e-drones",
		"https://www.olx.com.br/games",
		"https://www.olx.com.br/escritorio",
	}

	url := urls[rand.Intn(len(urls))]
	now := time.Now()

	filename := fmt.Sprintf("/tmp/olx-%s.html", now)

	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/109.0"

	pw, err := playwright.Run()
	if err != nil {
		log.Printf("could not run playwright\n")
		return ads, err
	}

	browser, err := pw.Chromium.Launch()
	if err != nil {
		log.Printf("could not launch chromium\n")
		return ads, err
	}

	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{UserAgent: &ua})
	if err != nil {
		log.Printf("could not create browser context\n")
		return ads, err
	}

	page, err := ctx.NewPage()
	if err != nil {
		log.Printf("could not create new browser tab\n")
		return ads, err
	}

	if _, err := page.Goto(url); err != nil {
		log.Printf("could not go to %s\n", url)
		return ads, err
	}

	html, err := page.Content()
	if err != nil {
		log.Printf("could not read content from page %s", url)
		return ads, err
	}

	err = os.WriteFile(filename, []byte(html), 0644)
	if err != nil {
		log.Printf("could not write file\n")
		return ads, err
	}

	if err = browser.Close(); err != nil {
		log.Printf("could not close browser\n")
		return ads, err
	}
	if err = pw.Stop(); err != nil {
		log.Printf("could not stop Playwright\n")
		return ads, err
	}

	t := &http.Transport{}
	t.RegisterProtocol("file", http.NewFileTransport(http.Dir("/")))

	c := colly.NewCollector()
	c.WithTransport(t)

	c.OnHTML(".olx-ad-card", func(e *colly.HTMLElement) {
		price := e.ChildText(".olx-ad-card__price")
		price = strings.TrimLeft(price, "R$ ")
		price = strings.ReplaceAll(price, ".", "")

		priceInt, err := strconv.Atoi(price)
		if err != nil {
			log.Printf("could not parse price: %s", err)
			return
		}

		ads = append(ads, OLXAd{
			Title:    e.ChildText(".olx-ad-card__title"),
			Price:    priceInt,
			Location: e.ChildTexts(".olx-ad-card__location-date-container>p")[0],
			Image:    e.ChildAttr(`source[type="image/jpeg"]`, "srcset"),
		})
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("Request URL: %s failed with response %v\nError: %v", r.Request.URL, r, err)
	})

	c.Visit("file://" + filename)
	c.Wait()

	return ads, err
}

func main() {
	var err error
	if os.Getenv("ENV") != "production" {
		err = godotenv.Load()
		if err != nil {
			log.Fatalf("could not load env file")
		}
	}

	if os.Getenv("ENV") == "production" {
		log.Printf("installing browsers...\n")
		err := playwright.Install()

		if err != nil {
			log.Fatalf("could not install browsers: %v", err)
		}

		log.Printf("browsers installed successfully\n")
	}

	db, err = sql.Open("libsql", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("failed to open db: %s", err)
	}

	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	var interval int
	interval, _ = strconv.Atoi(os.Getenv("INTERVAL"))

	if interval < 10 && os.Getenv("ENV") != "development" {
		log.Fatalf("Minimum interval is 10 minutes. Tried to run scraper with interval of %d minutes", interval)
	}

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Minute)

		for range ticker.C {
			log.Printf("fetching olx page\n")

			ads, err := randomPage()
			if err != nil {
				log.Printf("could not fetch ads: %v\n", err)
				continue
			}

			values := []any{}
			insert := "INSERT INTO olx_ads (title, price, image, location) VALUES "

			for _, ad := range ads {
				insert += "(?, ?, ?, ?),"
				values = append(values, ad.Title, ad.Price, ad.Image, ad.Location)
			}

			insert = strings.TrimRight(insert, ",")

			_, err = db.Exec(insert, values...)

			if err != nil {
				log.Printf("could not insert ads in database. ads: %v\nerror: %v\n", ads, err)
				continue
			}

			log.Printf("Successfully fetched ads.")
		}
	}()

	select {}
}
