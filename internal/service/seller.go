package service

type Seller struct {
	PortalID int64   `json:"id"`
	Title    string  `json:"title"`
	Orders   int     `json:"orders"`
	Reviews  int     `json:"reviews"`
	Rating   float32 `json:"rating"`
}
