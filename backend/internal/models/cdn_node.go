package models

type CDNNode struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Domain    string `json:"domain"`
	Protocol  string `json:"protocol"`
	Path      string `json:"path"`
	Sources   string `json:"sources"` // Список источников через \n
	Status    string `json:"status"`
}
