package service

import (
	"crypto/md5"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
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
	Fingerprint          string            `json:"-" db:"fingerprint"`
}

func (p *Product) calcFingerprint() {
	var sb strings.Builder
	if len(p.Characteristics) > 0 {
		for _, c := range p.Characteristics {
			sb.WriteString("character:")
			sb.WriteString(c.Title)
			if len(c.Values) == 0 {
				continue
			}
			sb.WriteString("|")

			for _, cv := range c.Values {
				sb.WriteString("char_value:")
				sb.WriteString(cv.Title)
				sb.WriteString("|")
				sb.WriteString(cv.Value)
			}
			sb.WriteString("|")
		}
	}

	if len(p.SkuList) > 0 {
		sb.WriteString("skuList:")
		for _, s := range p.SkuList {
			sb.WriteString(fmt.Sprintf("%d;%.2f;%.2f", s.AvailableAmount, s.FullPrice, s.PurchasePrice))
			sb.WriteString("|")
		}
	}

	productStr := fmt.Sprintf(
		"id:%d|descr:%s|rating:%.2f|orders:%d|avail:%d|sku:%s",
		p.PortalID, *p.Description, p.Rating, p.OrdersAmount, p.TotalAvailableAmount, sb.String(),
	)

	p.Fingerprint = fmt.Sprintf("%x", md5.Sum([]byte(productStr)))
}

func (p *Product) fingerprintExists(db *sqlx.DB) (bool, error) {
	var e int
	err := db.Get(&e, `SELECT 1 FROM products WHERE fingerprint = $1 AND id != $2 LIMIT 1`, p.Fingerprint, p.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (p *Product) remove(db *sqlx.DB) error {
	_, err := db.Exec(`DELETE FROM products WHERE id = $1`, p.ID)
	return err
}

func (p *Product) save(db *sqlx.DB) error {
	// check before
	var exist int
	// p.CategoryTitle = &p.Category.Title
	p.SellerID = &p.Seller.PortalID
	p.SellerTitle = &p.Seller.Title
	// Start a new transaction
	tx := db.MustBegin()
	if err := tx.Get(&exist, `SELECT 1 FROM products WHERE id = $1 LIMIT 1 FOR UPDATE NOWAIT`, p.ID); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("ERROR: NOWAIT: Product %d already parsed\n", p.ID)
			return nil
		}
		tx.Rollback()
		return err
	}

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
	  fingerprint = :fingerprint,
	  parsed_at = NOW() WHERE id = :id`
	if _, err := tx.NamedExec(query, p); err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

const batchSize = 100

type parseError struct {
	Error      error
	CategoryID int64
}

// CrawlProducts crawl all not parsed products
func CrawlProducts(db *sqlx.DB, rootCategoryID int64) error {
	var wg sync.WaitGroup

	leaves, err := categoryLeaves(db, rootCategoryID)
	if err != nil {
		return err
	}

	workerPoolSize := 100

	dataCh := make(chan int64, workerPoolSize)

	for i := 0; i < workerPoolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for categoryID := range dataCh {
				log.Printf("INFO: got category %d to parse\n", categoryID)
				if err := parseProducts(db, categoryID, batchSize); err != nil {
					log.Printf("ERROR: category %d, err: %v\n", categoryID, err)
					continue
				}
				log.Printf("INFO: category %d parsed successfully!\n", categoryID)
			}

		}()
	}

	for _, category := range leaves {
		categoryID := category.ID
		dataCh <- categoryID
	}
	close(dataCh)

	wg.Wait()

	return nil
}

// ParseProducts parses products from category
func parseProducts(db *sqlx.DB, categoryID int64, batchSize int64) error {
	var wg sync.WaitGroup

	minMaxIDQuery := `SELECT MIN(id) AS min_id, MAX(id) AS max_id FROM products
	  WHERE parsed_at IS NULL AND category_id = $1 LIMIT 1`

	minMaxID := &struct {
		MinID *int64 `db:"min_id"`
		MaxID *int64 `db:"max_id"`
	}{}

	if err := db.Get(minMaxID, minMaxIDQuery, categoryID); err != nil {
		if err == sql.ErrNoRows {
			log.Printf("INFO: No unparsed products for category #%d\n", categoryID)
			return nil

		}
		return err
	}

	if minMaxID.MinID == nil {
		return nil
	}

	workerPoolSize := 2
	dataCh := make(chan *Product, batchSize)

	for i := 0; i < workerPoolSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for product := range dataCh {
				var exist int
				if err := db.Get(&exist, `SELECT 1 FROM products WHERE id = $1 AND parsed_at IS NULL LIMIT 1`, product.ID); err != nil {
					if err == sql.ErrNoRows {
						log.Printf("ERROR: CHECKED: Product %d already parsed\n", product.ID)
						continue
					}
					log.Printf("ERROR: Load product failed: %v\n", err)
					continue
				}

				p, err := loadProduct(product.PortalID)
				if err != nil {
					log.Printf("ERROR: Load product failed: %v\n", err)
					continue
				}
				p.ID = product.ID
				log.Printf("INFO: Product loaded: %+v\n", p)

				p.calcFingerprint()
				log.Printf("INFO: Check fingerprint: %s\n", p.Fingerprint)
				exists, err := p.fingerprintExists(db)
				if err != nil {
					log.Printf("ERROR: Saving product failed: %v\n", err)
					continue
				}

				if exists {
					log.Printf("INFO: fingerprint exists: %s, remove duplicate\n", p.Fingerprint)
					if err := p.remove(db); err != nil {
						log.Printf("ERROR: Deleting product failed: %v\n", err)
					}
					continue
				}

				if err = p.save(db); err != nil {
					log.Printf("ERROR: Saving product failed: %v\n", err)
					continue
				}
				timeout := rand.Intn(2)
				log.Printf("Product %d has been parsed. Sleep %ds\n", product.ID, timeout)
				time.Sleep(time.Duration(timeout) * time.Second)
			}

		}()
	}

	query := `SELECT portal_id, id
	  FROM products
	  WHERE id BETWEEN $1 AND $2
	  AND session_id > 0
	  AND NOW()::date - to_timestamp(session_id / 1000000000)::date <= 1
	  AND parsed_at IS NULL
	  AND category_id = $3 ORDER BY id ASC`
	startBatch := *minMaxID.MinID

	log.Printf("Start parse products for category_id: %d, min_id: %d, max_id: %d\n",
		categoryID, startBatch, *minMaxID.MaxID)

	for startBatch <= *minMaxID.MaxID {
		nextID := startBatch + batchSize
		products := []*Product{}
		if err := db.Select(&products, query, startBatch, nextID, categoryID); err != nil {
			return err
		}

		for _, product := range products {
			dataCh <- product
		}

		startBatch = nextID
	}
	close(dataCh)

	wg.Wait()

	log.Printf("INFO: all products from category %d has been loaded\n", categoryID)

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
