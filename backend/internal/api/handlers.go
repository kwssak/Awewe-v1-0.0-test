package api

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
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
	db.Where("type = ?", "regular").Find(&nodes)
	return mgr.Restart(nodes)
}

func RegisterRoutes(r *gin.Engine, db *gorm.DB, mgr *xray.Manager, cfg *config.Config) {
	// Подписка
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

	// Логи Xray
	r.GET("/api/logs", auth(cfg.APIToken), func(c *gin.Context) {
		logFile := cfg.XrayConfig + ".log"
		f, err := os.Open(logFile)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"logs": "Xray not started yet or no log file"})
			return
		}
		defer f.Close()
		var lines []string
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if len(lines) > 100 {
			lines = lines[len(lines)-100:]
		}
		c.JSON(http.StatusOK, gin.H{"logs": strings.Join(lines, "\n")})
	})

	a := r.Group("", auth(cfg.APIToken))

	// Dashboard
	a.GET("/api/dashboard", func(c *gin.Context) {
		var nodeCount, clientCount, tariffCount int64
		db.Model(&models.Node{}).Count(&nodeCount)
		db.Model(&models.Client{}).Count(&clientCount)
		db.Model(&models.Tariff{}).Count(&tariffCount)
		c.JSON(http.StatusOK, gin.H{
			"nodes":   nodeCount,
			"clients": clientCount,
			"tariffs": tariffCount,
			"xray":    mgr.Running(),
			"port":    cfg.Port,
		})
	})

	// Nodes (обычные + автовыбор)
	a.GET("/api/nodes", func(c *gin.Context) {
		var nodes []models.Node
		db.Find(&nodes)
		out := []gin.H{}
		for _, n := range nodes {
			link := ""
			if n.Type == "regular" && n.CDNDomain != "" {
				link = cdnLink(n)
			}
			out = append(out, gin.H{"node": n, "link": link, "running": mgr.Running()})
		}
		c.JSON(http.StatusOK, out)
	})

	a.POST("/api/nodes", func(c *gin.Context) {
		var body struct {
			Name             string `json:"name"`
			Type             string `json:"type"` // "regular" или "auto"
			Protocol         string `json:"protocol"`
			Transport        string `json:"transport"`
			Encryption       string `json:"encryption"`
			Flow             string `json:"flow"`
			UTLS             string `json:"utls"`
			Limit            int    `json:"limit"`
			CDNDomain        string `json:"cdn_domain"`
			UplinkHTTPMethod string `json:"uplink_http_method"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}

		n := models.Node{
			Name:       body.Name,
			Type:       body.Type,
			Protocol:   body.Protocol,
			Transport:  body.Transport,
			Encryption: body.Encryption,
			Flow:       body.Flow,
			UTLS:       body.UTLS,
			Limit:      body.Limit,
			Status:     "active",
		}

		if body.Type == "regular" {
			n.UUID = utils.NewUUID()
			n.Port = nextPort(db)
			n.Path = "/" + utils.RandomHex(8)
			n.CDNDomain = body.CDNDomain
			n.UplinkHTTPMethod = body.UplinkHTTPMethod
			if n.UplinkHTTPMethod == "" {
				n.UplinkHTTPMethod = "HEAD"
			}
			n.IsCDN = body.CDNDomain != ""
		}

		if err := db.Create(&n).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Перезапускаем Xray только для regular нод
		if body.Type == "regular" {
			if err := reload(db, mgr); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "xray reload: " + err.Error()})
				return
			}
		}

		link := ""
		if body.Type == "regular" && body.CDNDomain != "" {
			link = cdnLink(n)
		}
		c.JSON(http.StatusOK, gin.H{"node": n, "link": link})
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
