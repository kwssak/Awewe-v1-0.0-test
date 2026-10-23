package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"awewe-panel/internal/config"
	"awewe-panel/internal/models"
	"awewe-panel/internal/utils"
	"awewe-panel/internal/xray"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, mgr *xray.Manager, cfg *config.Config) {
	// Публичный эндпоинт для проверки токена (используется при логине)
	r.GET("/api/dashboard", auth(cfg.APIToken), func(c *gin.Context) {
		var nodeCount, clientCount int64
		db.Model(&models.Node{}).Count(&nodeCount)
		db.Model(&models.Client{}).Count(&clientCount)
		c.JSON(http.StatusOK, gin.H{
			"nodes":   nodeCount,
			"clients": clientCount,
			"xray":    mgr.Running(),
			"port":    cfg.Port,
		})
	})

	a := r.Group("", auth(cfg.APIToken))

	// === NODES ===
	a.GET("/api/nodes", func(c *gin.Context) {
		var nodes []models.Node
		db.Find(&nodes)
		c.JSON(http.StatusOK, nodes)
	})

	a.POST("/api/nodes", func(c *gin.Context) {
		var body struct {
			Name      string `json:"name"`
			Type      string `json:"type"`
			CDNDomain string `json:"cdn_domain"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		n := models.Node{Name: body.Name, Type: body.Type, CDNDomain: body.CDNDomain, Status: "active"}
		if body.Type == "regular" {
			n.UUID = utils.NewUUID()
			n.Port = 2054 // Упрощено для примера, в проде нужен nextPort
			n.Path = "/" + utils.RandomHex(8)
			n.UplinkHTTPMethod = "HEAD"
		}
		db.Create(&n)
		if body.Type == "regular" {
			reload(db, mgr)
	 }
		c.JSON(http.StatusOK, gin.H{"node": n})
	})

	a.DELETE("/api/nodes/:id", func(c *gin.Context) {
		db.Delete(&models.Node{}, c.Param("id"))
		reload(db, mgr)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// === CDN NODES ===
	a.GET("/api/cdn-nodes", func(c *gin.Context) {
		var cdns []models.CDNNode
		db.Find(&cdns)
		c.JSON(http.StatusOK, cdns)
	})

	a.POST("/api/cdn-nodes", func(c *gin.Context) {
		var body models.CDNNode
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		body.Status = "active"
		db.Create(&body)
		c.JSON(http.StatusOK, body)
	})

	a.DELETE("/api/cdn-nodes/:id", func(c *gin.Context) {
		db.Delete(&models.CDNNode{}, c.Param("id"))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// === TARIFFS ===
	a.GET("/api/tariffs", func(c *gin.Context) {
		var tariffs []models.Tariff
		db.Find(&tariffs)
		c.JSON(http.StatusOK, tariffs)
	})

	a.POST("/api/tariffs", func(c *gin.Context) {
		var body models.Tariff
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
			return
		}
		db.Create(&body)
		c.JSON(http.StatusOK, body)
	})

	a.DELETE("/api/tariffs/:id", func(c *gin.Context) {
		db.Delete(&models.Tariff{}, c.Param("id"))
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// === SETTINGS ===
	a.GET("/api/settings", func(c *gin.Context) {
		data, _ := os.ReadFile(cfg.WebDir + "/settings.json")
		var settings map[string]string
		json.Unmarshal(data, &settings)
		if settings == nil { settings = make(map[string]string) }
		c.JSON(http.StatusOK, settings)
	})

	a.POST("/api/settings", func(c *gin.Context) {
		var body map[string]string
		c.ShouldBindJSON(&body)
		data, _ := json.Marshal(body)
		os.WriteFile(cfg.WebDir+"/settings.json", data, 0644)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func reload(db *gorm.DB, mgr *xray.Manager) error {
	var nodes []models.Node
	db.Where("type = ?", "regular").Find(&nodes)
	return mgr.Restart(nodes)
}

func auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Token") != token {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		c.Next()
	}
}
