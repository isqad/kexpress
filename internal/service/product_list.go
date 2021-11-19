package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
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
	PortalID   int64     `json:"productId" db:"portal_id"`
	Title      string    `json:"title" db:"title"`
	CategoryID int64     `json:"categoryId" db:"category_id"`
	Rating     float32   `json:"rating" db:"rating"`
	CreatedAt  time.Time `json:"-" db:"created_at"`
	SellPrice  float32   `json:"sellPrice" db:"-"`
}

func (p *ProductListResponse) SaveProducts(db *sqlx.DB) error {
	if p.Error != "" {
		return errors.New(p.Error)
	}

	_, err := db.NamedExec(`INSERT INTO products (portal_id, title, category_id, rating, created_at)
	  VALUES (:portal_id, :title, :category_id, :rating, NOW())`, p.Payload.Products)

	return err
}

var (
	totalPages, totalProducts int
)

func LoadProductList(db *sqlx.DB, page int, categoryID int) error {
	log.Printf("Parse page: %d\n", page)

	if totalPages > 0 && page > totalPages {
		log.Println("All pages parsed and saved!")
		return nil
	}

	url := fmt.Sprintf("https://api.kazanexpress.ru/api/v2/main/search/product?size=100&page=%d&categoryId=%d&sortBy=orders&order=descending", page, categoryID)
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

	if totalProducts == 0 {
		totalProducts = pResponse.Payload.TotalProducts
		totalPages = int(math.Ceil(float64(totalProducts) / float64(100)))
		log.Printf("Total products: %d, Total pages: %d\n", totalProducts, totalPages)
	}

	err = pResponse.SaveProducts(db)
	if err != nil {
		return err
	}
	timeout := rand.Intn(3)
	log.Printf("Page %d has been parsed. Sleep %ds\n", page, timeout)
	time.Sleep(time.Duration(timeout) * time.Second)

	return LoadProductList(db, page+1, categoryID)
}
