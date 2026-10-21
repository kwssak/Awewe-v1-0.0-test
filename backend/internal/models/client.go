package models

type Client struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	NodeID uint   `json:"node_id"`
	Email  string `json:"email"`
	UUID   string `json:"uuid"`
}
