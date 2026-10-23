package models

type Tariff struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Days     int    `json:"days"`
	Traffic  int    `json:"traffic"`
	Devices  int    `json:"devices"`
	Speed    int    `json:"speed"`
	Desc     string `json:"desc"`
}
