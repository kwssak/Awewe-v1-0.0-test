package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"awewe-panel/internal/api"
	"awewe-panel/internal/config"
	"awewe-panel/internal/database"
	"awewe-panel/internal/models"
	"awewe-panel/internal/xray"
)

func main() {
	cfgPath := flag.String("config", "/etc/awewe-panel/config.json", "config file")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if cfg.Port < 1024 || cfg.Port > 60000 {
		log.Fatalf("port must be between 1024 and 60000, got %d", cfg.Port)
	}

	if err := database.Init(cfg.DBPath); err != nil {
		log.Fatalf("db: %v", err)
	}

	mgr := xray.New(cfg.XrayBin, cfg.XrayConfig)

	var nodes []models.Node
	database.DB.Find(&nodes)
	if len(nodes) > 0 {
		if err := mgr.Restart(nodes); err != nil {
			log.Printf("xray start: %v", err)
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	api.RegisterRoutes(r, database.DB, mgr, cfg)

	r.Static("/ui", cfg.WebDir)
	r.GET(cfg.WebPath, func(c *gin.Context) { c.Redirect(302, "/ui/") })
	r.GET("/", func(c *gin.Context) { c.Redirect(302, cfg.WebPath) })

	log.Printf("Awewe Panel on :%d%s", cfg.Port, cfg.WebPath)
	if err := r.Run(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("server: %v", err)
	}
}
