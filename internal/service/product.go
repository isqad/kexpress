package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ProductResponsePayload is payload of reponse
type ProductResponsePayload struct {
	Data      Product `json:"data"`
	Promotion string  `json:"promotion"`
}

// ProductResponse is raw response from server
type ProductResponse struct {
	Error   string                 `json:"error"`
	Payload ProductResponsePayload `json:"payload"`
}

// Product is product
type Product struct {
	PortalID             int64     `json:"id" db:"portal_id"`
	Title                string    `json:"title" db:"title"`
	Description          string    `json:"description" db:"description"`
	CategoryID           int64     `json:"-" db:"category_id"`
	CategoryTitle        string    `json:"-" db:"category_title"`
	Category             *Category `json:"category" db:"-"`
	SellerID             int64     `json:"-" db:"seller_id"`
	SellerTitle          string    `json:"-" db:"seller_title"`
	Seller               *Seller   `json:"seller" db:"-"`
	OrdersAmount         int       `json:"ordersAmount" db:"orders_amount"`
	ReviewsAmount        int       `json:"reviewsAmount" db:"reviews_amount"`
	TotalAvailableAmount int       `json:"totalAvailableAmount" db:"total_available_amount"`
	Rating               float32   `json:"rating" db:"rating"`
	CreatedAt            time.Time `json:"-" db:"created_at"`
	RawJSON              string    `json:"-" db:"raw_json"`
	SellPrice            float32   `json:"sellPrice" db:"-"`
}

// LoadProduct temporary func
func LoadProduct(id int) error {
	url := fmt.Sprintf("https://api.kazanexpress.ru/api/v2/product/%d", id)
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	req, err := newRequest(url)
	if err != nil {
		return err
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	p := &ProductResponse{}

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(p)
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", p)

	return nil
}
