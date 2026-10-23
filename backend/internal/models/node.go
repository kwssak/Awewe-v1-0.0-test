package models

type Node struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"` // "regular" или "auto"
	Protocol         string `json:"protocol"`
	Transport        string `json:"transport"`
	Encryption       string `json:"encryption"`
	Flow             string `json:"flow"`
	UTLS             string `json:"utls"`
	Limit            int    `json:"limit"`
	Status           string `json:"status"`
	
	// Только для regular нод
	Port             int    `json:"port,omitempty"`
	UUID             string `json:"uuid,omitempty"`
	Path             string `json:"path,omitempty"`
	UplinkHTTPMethod string `json:"uplink_http_method,omitempty"`
	IsCDN            bool   `json:"is_cdn,omitempty"`
	CDNDomain        string `json:"cdn_domain,omitempty"`
}
