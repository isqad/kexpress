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

const perPage = 24

// ProductListResponse is response from API
type ProductListResponse struct {
	Error   string                     `json:"error"`
	Payload *ProductListReponsePayload `json:"payload"`
}

// ProductListReponsePayload is payload of response from API
type ProductListReponsePayload struct {
	TotalProducts int              `json:"totalProducts"`
	Products      []*ProductOfList `json:"products"`
	AdultContent  bool             `json:"adultContent"`
}

// ProductOfList is item from list
type ProductOfList struct {
	PortalID         int64     `json:"productId" db:"portal_id"`
	Title            string    `json:"title" db:"title"`
	CategoryID       int64     `json:"-" db:"category_id"`
	PortalCategoryID int64     `json:"categoryId" db:"portal_category_id"`
	Rating           float32   `json:"rating" db:"rating"`
	CreatedAt        time.Time `json:"-" db:"created_at"`
	SellPrice        float32   `json:"sellPrice" db:"-"`
	SessionID        int64     `json:"-" db:"session_id"`
}

func (p *ProductListResponse) saveProducts(db *sqlx.DB) error {
	if p.Error != "" {
		return errors.New(p.Error)
	}

	tx := db.MustBegin()
	for _, p := range p.Payload.Products {
		_, err := tx.NamedExec(`INSERT INTO products (portal_id, title, portal_category_id, category_id, rating, session_id, created_at)
		VALUES (:portal_id, :title, :portal_category_id, :category_id, :rating, :session_id, NOW())
		ON CONFLICT ON CONSTRAINT uniq_portal_id_session_id_products DO NOTHING`, p)

		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// CrawlProductList crawls product listings
func CrawlProductList(db *sqlx.DB, rootCategoryID int64) error {
	sessionID := time.Now().UnixNano()
	var wg sync.WaitGroup
	workerPoolSize := 100

	dataCh := make(chan *Category, workerPoolSize)

	leaves, err := categoryLeaves(db, rootCategoryID)
	if err != nil {
		return err
	}
	for i := 0; i < workerPoolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for category := range dataCh {
				log.Printf("INFO: got category %d to load\n", category.ID)
				cid := category.ID
				pid := category.PortalID
				amount := category.ProductAmount
				totalProducts := amount
				totalPages := int(math.Ceil(float64(totalProducts) / float64(perPage)))
				log.Printf("INFO: category: %d, Total products: %d, Total pages: %d\n", cid, totalProducts, totalPages)

				if err := loadProductList(db, sessionID, 0, pid, cid, totalPages); err != nil {
					log.Printf("ERROR: Error loading product list for category: #%d\n", cid)
					return
				}
				log.Printf("INFO: category %d loaded successfully!\n", category.ID)
			}
		}()
	}

	for _, category := range leaves {
		dataCh <- category
	}
	close(dataCh)

	wg.Wait()

	return nil
}

func loadProductList(db *sqlx.DB, sessID int64, page int, portalCategoryID int64, categoryID int64, totalPages int) error {
	log.Printf("Parse listing page: %d\n", page)

	if totalPages > 0 && page > totalPages {
		log.Println("All pages parsed and saved!")
		return nil
	}

	url := fmt.Sprintf("https://api.kazanexpress.ru/api/v2/main/search/product?size=%d&page=%d&categoryId=%d&sortBy=orders&order=descending", perPage, page, portalCategoryID)
	client := &http.Client{
		Timeout: 120 * time.Second,
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
		p.SessionID = sessID
	}

	err = pResponse.saveProducts(db)
	if err != nil {
		return err
	}
	timeout := rand.Intn(7)
	log.Printf("Page %d has been parsed. Sleep %ds\n", page, timeout)
	time.Sleep(time.Duration(timeout) * time.Second)

	return loadProductList(db, sessID, page+1, portalCategoryID, categoryID, totalPages)
}
