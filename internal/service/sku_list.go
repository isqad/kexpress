package service

import "github.com/jmoiron/sqlx"

// Sku is a stock keeping unit
type Sku struct {
	ID              int64                `json:"-" db:"id"`
	ProductID       int64                `json:"-" db:"product_id"`
	CharValueID     *int64               `json:"-" db:"char_value_id"`
	AvailableAmount int                  `json:"availableAmount" db:"available_amount"`
	FullPrice       float32              `json:"fullPrice" db:"full_price"`
	PurchasePrice   float32              `json:"purchasePrice" db:"purchase_price"`
	Characteristics []*SkuCharacteristic `json:"characteristics" db:"-"`
}

func (sku *Sku) save(tx *sqlx.Tx) error {
	return tx.Get(&sku.ID, `
	INSERT INTO skus (product_id, available_amount, full_price, purchase_price, created_at)
	  VALUES ($1, $2, $3, $4, NOW()) RETURNING id`,
		sku.ProductID,
		sku.AvailableAmount,
		sku.FullPrice,
		sku.PurchasePrice,
	)
}

// SkuCharacteristic keeps coordinates for characteristic of SKU
type SkuCharacteristic struct {
	CharIndex  int `json:"charIndex"`
	ValueIndex int `json:"valueIndex"`
}
