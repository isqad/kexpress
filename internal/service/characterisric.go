package service

import (
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// Characteristic is something charaterizing the product
type Characteristic struct {
	ID        int64        `json:"-" db:"id"`
	Title     string       `json:"title" db:"title"`
	Values    []*CharValue `json:"values" db:"-"`
	CreatedAt time.Time    `json:"-" db:"created_at"`
}

func (c *Characteristic) save(tx *sqlx.Tx) error {
	query := `INSERT INTO characteristics (title, created_at) VALUES ($1, NOW())
	  ON CONFLICT ON CONSTRAINT uniq_title_characteristics DO NOTHING RETURNING id`
	if err := tx.Get(&c.ID, query, c.Title); err != nil {
		if err == sql.ErrNoRows {
			if err = tx.Get(&c.ID, `SELECT id FROM characteristics WHERE title = $1`, c.Title); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}

// CharValue value of the Characteristic
type CharValue struct {
	ID     int64  `json:"-" db:"id"`
	CharID int64  `json:"-" db:"char_id"`
	Title  string `json:"title"`
	Value  string `json:"value"`
}

func (cv *CharValue) save(tx *sqlx.Tx) error {
	query := `INSERT INTO char_values
    (title, value, char_id, created_at) VALUES ($1, $2, $3, NOW())
    ON CONFLICT ON CONSTRAINT uniq_char_id_title_value DO NOTHING RETURNING id`
	if err := tx.Get(&cv.ID, query, cv.Title, cv.Value, cv.CharID); err != nil {
		if err == sql.ErrNoRows {
			if err = tx.Get(&cv.ID,
				`SELECT id FROM char_values WHERE title = $1 AND value = $2 AND char_id = $3`,
				cv.Title, cv.Value, cv.CharID); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
