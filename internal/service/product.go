package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// ProductResponsePayload is payload of reponse
type ProductResponsePayload struct {
	Data      *Product `json:"data"`
	Promotion string   `json:"promotion"`
}

// ProductResponse is raw response from server
type ProductResponse struct {
	Error   string                 `json:"error"`
	Payload ProductResponsePayload `json:"payload"`
}

// Product is product
type Product struct {
	ID                   int64             `json:"-" db:"id"`
	PortalID             int64             `json:"id" db:"portal_id"`
	Title                string            `json:"title" db:"title"`
	Description          *string           `json:"description" db:"description"`
	PortalCategoryID     *int64            `json:"-" db:"portal_category_id"`
	CategoryID           int64             `json:"-" db:"category_id"`
	CategoryTitle        *string           `json:"-" db:"category_title"`
	Category             *Category         `json:"category" db:"-"`
	SellerID             *int64            `json:"-" db:"seller_id"`
	SellerTitle          *string           `json:"-" db:"seller_title"`
	Seller               *Seller           `json:"seller" db:"-"`
	OrdersAmount         int               `json:"ordersAmount" db:"orders_amount"`
	ReviewsAmount        int               `json:"reviewsAmount" db:"reviews_amount"`
	TotalAvailableAmount int               `json:"totalAvailableAmount" db:"total_available_amount"`
	Rating               float32           `json:"rating" db:"rating"`
	CreatedAt            time.Time         `json:"-" db:"created_at"`
	Characteristics      []*Characteristic `json:"characteristics" db:"-"`
	SkuList              []*Sku            `json:"skuList" db:"-"`
}

func (p *Product) save(db *sqlx.DB) error {
	p.CategoryTitle = &p.Category.Title
	p.SellerID = &p.Seller.PortalID
	p.SellerTitle = &p.Seller.Title
	// Start a new transaction
	tx := db.MustBegin()
	charValues := make(map[int]map[int]int64)

	if len(p.Characteristics) > 0 {
		// save chars
		log.Printf("Saving characteristics for product: #%d %s\n", p.PortalID, p.Title)
		for ic, c := range p.Characteristics {

			charValues[ic] = make(map[int]int64)

			if err := c.save(tx); err != nil {
				tx.Rollback()
				return err
			}
			if len(c.Values) == 0 {
				log.Printf("No Values given for characteristic: %s\n", c.Title)
				continue
			}
			log.Printf("Saving characteristic values for product: #%d %s\n", p.PortalID, p.Title)
			for icv, cv := range c.Values {
				cv.CharID = c.ID
				if err := cv.save(tx); err != nil {
					tx.Rollback()
					return err
				}
				charValues[ic][icv] = cv.ID
			}
		}
		log.Printf("Gathered charValues: %+v\n", charValues)
	} else {
		log.Printf("No Characteristics given for product: #%d %s\n", p.PortalID, p.Title)
	}

	// save sku list
	if len(p.SkuList) > 0 {
		log.Printf("Saving skuList for product: #%d %s\n", p.PortalID, p.Title)
		for _, s := range p.SkuList {
			s.ProductID = p.ID

			if err := s.save(tx); err != nil {
				tx.Rollback()
				return err
			}

			if len(s.Characteristics) > 0 {
				for _, ch := range s.Characteristics {
					if v, ok := charValues[ch.CharIndex][ch.ValueIndex]; ok {
						skuCharValue := &SkuCharValue{
							SkuID:       s.ID,
							CharValueID: v,
						}
						if err := skuCharValue.save(tx); err != nil {
							tx.Rollback()
							return err
						}
					}
				}
			}
		}
	} else {
		log.Printf("No SkuList given for product: #%d %s\n", p.PortalID, p.Title)
	}

	query := `UPDATE products SET
	  seller_id = :seller_id,
	  orders_amount = :orders_amount,
	  reviews_amount = :reviews_amount,
	  total_available_amount = :total_available_amount,
	  category_title = :category_title,
	  seller_title = :seller_title,
	  description = :description,
	  parsed_at = NOW() WHERE id = :id`
	if _, err := tx.NamedExec(query, p); err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

const batchSize = 1000

// CrawlProducts crawl all not parsed products
func CrawlProducts(db *sqlx.DB, rootCategoryID int64) error {
	var wg sync.WaitGroup

	leaves, err := categoryLeaves(db, rootCategoryID)
	if err != nil {
		return err
	}

	for _, category := range leaves {
		wg.Add(1)

		categoryID := category.ID
		go func() {
			defer wg.Done()

			timeout := rand.Intn(15)
			log.Printf("Random sleep for avoid DDoS on %ds before start\n", timeout)
			time.Sleep(time.Duration(timeout) * time.Second)

			if err := parseProducts(db, categoryID, batchSize); err != nil {
				log.Printf("ERROR: parsing category %d failed: %v\n", categoryID, err)
			}

			log.Printf("INFO: category %d parsed successfully!\n", categoryID)
		}()

	}

	wg.Wait()

	return nil
}

// ParseProducts parses products from category
func parseProducts(db *sqlx.DB, categoryID int64, batchSize int64) error {
	minMaxIDQuery := `SELECT MIN(id) AS min_id, MAX(id) AS max_id FROM products
	  WHERE parsed_at IS NULL AND category_id = $1`

	minMaxID := &struct {
		MinID *int64 `db:"min_id"`
		MaxID *int64 `db:"max_id"`
	}{}

	if err := db.Get(minMaxID, minMaxIDQuery, categoryID); err != nil {
		return err
	}

	if minMaxID.MinID == nil {
		log.Printf("INFO: No unparsed products for category #%d\n", categoryID)
		return nil
	}

	log.Printf("Start parse products for category_id: %d, min_id: %d, max_id: %d\n",
		categoryID, minMaxID.MinID, minMaxID.MaxID)

	query := `SELECT portal_id, id FROM products WHERE id BETWEEN $1 AND $2 AND parsed_at IS NULL AND category_id = $3`
	products := []*Product{}
	startBatch := *minMaxID.MinID

	for startBatch <= *minMaxID.MaxID {
		nextID := startBatch + batchSize
		if err := db.Select(&products, query, startBatch, nextID, categoryID); err != nil {
			return err
		}

		// Handle batch
		// TODO: use channels and goroutines
		for _, product := range products {
			p, err := loadProduct(product.PortalID)
			if err != nil {
				log.Printf("Load product failed: %v\n", err)
				continue
			}
			p.ID = product.ID
			log.Printf("Product loaded: %+v\n", p)
			if err = p.save(db); err != nil {
				log.Printf("Saving product failed: %v\n", err)
				continue
			}
			// We will respect the server
			timeout := rand.Intn(7)
			log.Printf("Product %s has been parsed. Sleep %ds\n", p.Title, timeout)
			time.Sleep(time.Duration(timeout) * time.Second)
		}

		startBatch = nextID
	}

	return nil
}

func loadProduct(portalID int64) (*Product, error) {
	url := fmt.Sprintf("https://api.kazanexpress.ru/api/v2/product/%d", portalID)
	client := &http.Client{
		Timeout: 120 * time.Second,
	}
	req, err := newRequest(url)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	p := &ProductResponse{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(p)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()

	if p.Error != "" {
		return nil, errors.New(p.Error)
	}

	return p.Payload.Data, nil
}
