package service

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jmoiron/sqlx"
)

const url = "https://api.kazanexpress.ru/api/v2/main/search/category?&categoryId=1"

// Category is category of products
type Category struct {
	ID            int64        `json:"-" db:"id"`
	PortalID      int64        `json:"id" db:"portal_id"`
	Title         string       `json:"title" db:"title"`
	Parent        *Category    `json:"-" db:"-"`
	ParentID      int64        `json:"-" db:"parent_id"`
	ProductAmount int          `json:"productAmount,omitempty" db:"products_amount"`
	Children      []*Category  `json:"children" db:"-"`
	CreatedAt     time.Time    `json:"-" db:"created_at"`
	UpdatedAt     time.Time    `json:"-" db:"updated_at"`
	History       pgtype.JSONB `json:"-" db:"history"`
}

// CategoryResponse is response from server
type CategoryResponse struct {
	Error   string                   `json:"error"`
	Payload *CategoryResponsePayload `json:"payload"`
}

// CategoryResponsePayload is response payload
type CategoryResponsePayload struct {
	RootCategory *Category `json:"category"`
}

// RootCategories returns all root categories
func RootCategories(db *sqlx.DB) ([]*Category, error) {
	query := `SELECT * FROM categories WHERE parent_id = 0`
	roots := []*Category{}
	err := db.Select(&roots, query)
	if err != nil {
		return nil, err
	}

	return roots, nil
}

func findCategory(db *sqlx.DB, ID int64) (*Category, error) {
	c := &Category{}
	if err := db.Get(c, `SELECT * FROM categories WHERE id = $1 LIMIT 1`, ID); err != nil {
		return nil, err
	}
	return c, nil
}

// CategoryLeaves fetches leaves
func categoryLeaves(db *sqlx.DB, rootCategoryID int64) ([]*Category, error) {
	query := `WITH RECURSIVE t AS (
				SELECT id,
				       title,
					   products_amount,
					   parent_id,
					   portal_id,
					   NOT EXISTS (SELECT NULL FROM categories cl WHERE categories.id = cl.parent_id) is_leaf FROM categories WHERE id = $1
				UNION ALL
				SELECT categories.id,
				       categories.title,
					   categories.products_amount,
				       categories.parent_id,
					   categories.portal_id,
					   NOT EXISTS (SELECT NULL FROM categories cl WHERE categories.id = cl.parent_id) is_leaf FROM t JOIN categories ON t.id = categories.parent_id)
			  SELECT id, title, products_amount, portal_id FROM t WHERE is_leaf ORDER BY products_amount`
	leaves := []*Category{}
	err := db.Select(&leaves, query, rootCategoryID)
	if err != nil {
		return nil, err
	}

	return leaves, nil
}

func loadCategories() ([]*Category, error) {
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

	return r.Payload.RootCategory.Children, nil
}

func saveCategories(db *sqlx.DB, children []*Category) error {
	var id int64
	for _, c := range children {
		log.Printf("Save category: %s\n", c.Title)

		if err := db.Get(&id, `INSERT INTO categories (portal_id, title, products_amount, parent_id, created_at)
		  VALUES ($1, $2, $3, $4, NOW())
		  ON CONFLICT ON CONSTRAINT uniq_portal_id_categories DO UPDATE
		    SET updated_at = NOW(),
			  history =
			    categories.history ||
				  ('{"' || extract(epoch from now()) || '":' ||
					'{"products_amount_was":' || categories.products_amount ||
				  ', "products_amount_new":' || EXCLUDED.products_amount || '}}')::jsonb,
			  products_amount = EXCLUDED.products_amount
			WHERE categories.products_amount != EXCLUDED.products_amount
		  RETURNING id`,
			c.PortalID,
			c.Title,
			c.ProductAmount,
			c.ParentID,
		); err != nil {
			if err == sql.ErrNoRows {
				if err = db.Get(&id, `SELECT id FROM categories WHERE portal_id = $1`, c.PortalID); err != nil {
					return err
				}
			} else {
				return err
			}
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
