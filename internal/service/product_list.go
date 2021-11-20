package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type ProductListResponse struct {
	Error   string                     `json:"error"`
	Payload *ProductListReponsePayload `json:"payload"`
}

type ProductListReponsePayload struct {
	TotalProducts int              `json:"totalProducts"`
	Products      []*ProductOfList `json:"products"`
	AdultContent  bool             `json:"adultContent"`
}

type ProductOfList struct {
	PortalID         int64     `json:"productId" db:"portal_id"`
	Title            string    `json:"title" db:"title"`
	CategoryID       int64     `json:"-" db:"category_id"`
	PortalCategoryID int64     `json:"categoryId" db:"portal_category_id"`
	Rating           float32   `json:"rating" db:"rating"`
	CreatedAt        time.Time `json:"-" db:"created_at"`
	SellPrice        float32   `json:"sellPrice" db:"-"`
}

func (p *ProductListResponse) SaveProducts(db *sqlx.DB) error {
	if p.Error != "" {
		return errors.New(p.Error)
	}

	_, err := db.NamedExec(`INSERT INTO products (portal_id, title, portal_category_id, category_id, rating, created_at)
VALUES (:portal_id, :title, :portal_category_id, :category_id, :rating, NOW())`, p.Payload.Products)

	return err
}

const perPage = 24

var (
	totalPages, totalProducts int
)

// CrawlProductList crawls product listings
func CrawlProductList(db *sqlx.DB, rootCategoryID int64) error {
	var wg sync.WaitGroup

	// Fetch leaves
	leaves, err := categoryLeaves(db, rootCategoryID)
	if err != nil {
		return err
	}

	for _, category := range leaves {
		wg.Add(1)

		c := category

		go func() {
			defer wg.Done()
			log.Printf("Crawl category %d: %s Products amount: %d\n", c.PortalID, c.Title, c.ProductAmount)

			totalProducts = c.ProductAmount
			totalPages = int(math.Ceil(float64(totalProducts) / float64(perPage)))
			log.Printf("Total products: %d, Total pages: %d\n", totalProducts, totalPages)

			if err = loadProductList(db, 0, c.PortalID, c.ID); err != nil {
				log.Printf("ERROR: Error loading product list for category: #%d\n", c.ID)
				return
			}
			log.Printf("Category %d: %s has been parsed\n", c.PortalID, c.Title)
		}()
	}

	wg.Wait()

	return nil
}

func loadProductList(db *sqlx.DB, page int, portalCategoryID int64, categoryID int64) error {
	log.Printf("Parse listing page: %d\n", page)

	if totalPages > 0 && page > totalPages {
		log.Println("All pages parsed and saved!")
		return nil
	}

	url := fmt.Sprintf("https://api.kazanexpress.ru/api/v2/main/search/product?size=%d&page=%d&categoryId=%d&sortBy=orders&order=descending", perPage, page, portalCategoryID)
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	req, err := newRequest(url)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	pResponse := &ProductListResponse{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(pResponse)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if len(pResponse.Payload.Products) == 0 {
		return nil
	}
	for _, p := range pResponse.Payload.Products {
		p.CategoryID = categoryID
	}

	err = pResponse.SaveProducts(db)
	if err != nil {
		return err
	}
	timeout := rand.Intn(15)
	log.Printf("Page %d has been parsed. Sleep %ds\n", page, timeout)
	time.Sleep(time.Duration(timeout) * time.Second)

	return loadProductList(db, page+1, portalCategoryID, categoryID)
}
