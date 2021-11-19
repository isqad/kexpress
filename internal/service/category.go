package service

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jmoiron/sqlx"
)

const url = "https://api.kazanexpress.ru/api/main/root-categories?eco=false"

type Category struct {
	PortalID      int64       `json:"id" db:"portal_id"`
	Title         string      `json:"title" db:"title"`
	Parent        *Category   `json:"-" db:"-"`
	ParentID      int64       `json:"-" db:"parent_id"`
	ProductAmount int         `json:"productAmount,omitempty" db:"products_amount"`
	Children      []*Category `json:"children" db:"-"`
	CreatedAt     time.Time   `json:"-" db:"created_at"`
	UpdatedAt     time.Time   `json:"-" db:"updated_at"`
}

type CategoryResponse struct {
	Error   string      `json:"error"`
	Payload []*Category `json:"payload"`
}

func loadCategories() ([]*Category, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := newRequest(url)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	r := &CategoryResponse{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(r)
	if err != nil {
		return nil, err
	}

	if r.Error != "" {
		return nil, errors.New(r.Error)
	}
	log.Println("Categories has been loaded")

	return r.Payload, nil
}

func saveCategories(db *sqlx.DB, children []*Category) error {
	var id int64
	for _, c := range children {
		log.Printf("Save category: %s\n", c.Title)

		if err := db.Get(&id, `INSERT INTO categories (portal_id, title, products_amount, parent_id, created_at)
		  VALUES ($1, $2, $3, $4, NOW())
		  ON CONFLICT ON CONSTRAINT uniq_portal_id_categories DO UPDATE
		    SET updated_at = NOW(),
			  products_amount = EXCLUDED.products_amount
		  RETURNING id`,
			c.PortalID,
			c.Title,
			c.ProductAmount,
			c.ParentID,
		); err != nil {
			return err
		}

		if c.Children != nil {
			log.Println("Category has children")
			for _, cc := range c.Children {
				cc.ParentID = id
			}
			if err := saveCategories(db, c.Children); err != nil {
				return err
			}
		}
	}
	log.Println("Categories saved")

	return nil
}

func CrawlCategories(db *sqlx.DB) error {
	log.Println("Crawl categories")
	c, err := loadCategories()
	if err != nil {
		return err
	}
	return saveCategories(db, c)
}
