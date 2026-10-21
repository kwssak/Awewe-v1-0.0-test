package api

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"awewe-panel/internal/config"
	"awewe-panel/internal/models"
	"awewe-panel/internal/utils"
	"awewe-panel/internal/xray"
)

func cdnLink(n models.Node) string {
	q := url.Values{}
	q.Set("encryption", "none")
	q.Set("security", "tls")
	q.Set("type", "xhttp")
	q.Set("path", n.Path)
	q.Set("sni", n.CDNDomain)
	q.Set("fp", "chrome")
	q.Set("mode", "auto")
	q.Set("uplinkHttpMethod", n.UplinkHTTPMethod)
	return fmt.Sprintf("vless://%s@%s:443?%s#%s", n.UUID, n.CDNDomain, q.Encode(), url.QueryEscape(n.Name))
}

func nextPort(db *gorm.DB) int {
	var maxPort int
	db.Model(&models.Node{}).Select("COALESCE(MAX(port),2053)").Scan(&maxPort)
	return maxPort + 1
}

func reload(db *gorm.DB, mgr *xray.Manager) error {
	var nodes []models.Node
	db.Find(&nodes)
	return mgr.Restart(nodes)
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, mgr *xray.Manager, cfg *config.Config) {
	r.GET(cfg.SubBase+"/:email", func(c *gin.Context) {
		email := c.Param("email")
		var clients []models.Client
		db.Where("email = ?", email).Find(&clients)
		links := []string{}
		for _, cl := range clients {
			var n models.Node
			if db.First(&n, cl.NodeID).Error == nil {
				links = append(links, cdnLink(n))
			}
		}
		c.Header("Content-Type", "text/plain")
		c.String(http.StatusOK, base64.StdEncoding.EncodeToString([]byte(strings.Join(links, "\n"))))
	})

	a := r.Group("", auth(cfg.APIToken))

	a.GET("/api/nodes", func(c *gin.Context) {
		var nodes []models.Node
		db.Find(&nodes)
		out := []gin.H{}
		for _, n := range nodes {
			out = append(out, gin.H{"node": n, "link": cdnLink(n), "running": mgr.Running()})
		}
		c.JSON(http.StatusOK, out)
	})

	a.POST("/api/nodes/cdn", func(c *gin.Context) {
		var body struct {
			Name             string `json:"name"`
			CDNDomain        string `json:"cdn_domain"`
			UplinkHTTPMethod string `json:"uplink_http_method"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Name == "" || body.CDNDomain == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name and cdn_domain required"})
			return
		}
		if body.UplinkHTTPMethod == "" {
			body.UplinkHTTPMethod = "HEAD"
		}
		n := models.Node{
			Name:             body.Name,
			UUID:             utils.NewUUID(),
			Port:             nextPort(db),
			Path:             "/" + utils.RandomHex(8),
			UplinkHTTPMethod: body.UplinkHTTPMethod,
			IsCDN:            true,
			CDNDomain:        body.CDNDomain,
		}
		if err := db.Create(&n).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := reload(db, mgr); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "xray reload: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"node": n, "link": cdnLink(n)})
	})

	a.DELETE("/api/nodes/:id", func(c *gin.Context) {
		id := c.Param("id")
		if err := db.Delete(&models.Node{}, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = reload(db, mgr)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	a.POST("/api/clients", func(c *gin.Context) {
		var body struct {
			NodeID uint   `json:"node_id"`
			Email  string `json:"email"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.NodeID == 0 || body.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "node_id and email required"})
			return
		}
		cl := models.Client{NodeID: body.NodeID, Email: body.Email, UUID: utils.NewUUID()}
		if err := db.Create(&cl).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, cl)
	})

	a.GET("/api/clients", func(c *gin.Context) {
		var clients []models.Client
		db.Find(&clients)
		c.JSON(http.StatusOK, clients)
	})
}
