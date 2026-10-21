package models

type Node struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	UUID             string `json:"uuid"`
	Port             int    `json:"port"`
	Path             string `json:"path"`
	UplinkHTTPMethod string `json:"uplink_http_method"`
	IsCDN            bool   `json:"is_cdn"`
	CDNDomain        string `json:"cdn_domain"`
}
