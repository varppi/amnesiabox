package server

import (
	"amnesiabox/internal/config"
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"github.com/psanford/memfs"
)

var (
	Config   *config.Configuration
	srv      *http.Server
	stopChan = make(chan *struct{}, 1)

	//go:embed views/*
	dirFS              embed.FS
	serverTemplates, _ = template.ParseFS(dirFS, "views/*.html")
	sites              = make(map[string]*memfs.FS)
	keys               = make(map[string]string)
	router             *gin.Engine
)

func StartServer(_config *config.Configuration) (chan *struct{}, error) {
	Config = _config

	gin.SetMode("release")
	router = gin.New()
	router.Use(gin.Recovery())

	// Routes
	router.GET("/admin", admin)
	router.GET("/dashboard", dashboard)
	router.GET("/", homepage)
	router.GET("/sites/*s", siteHandler)
	router.POST("/upload", upload)
	router.POST("/dashboard/update", updateSite)
	router.POST("/dashboard/delete", deleteSite)
	router.POST("/dashboard/logout", logOut)
	router.POST("/admin/delete", adminDelete)
	router.POST("/login", login)
	router.Any("/captcha/*w", gin.WrapH(captcha.Server(captcha.StdWidth/2, 40)))
	router.StaticFS("/public", http.Dir("public"))

	srv = &http.Server{
		Addr:    Config.Listener,
		Handler: router.Handler(),
	}
	log.Printf("starting listener on %s\n", Config.Listener)
	log.Printf("ADMIN KEY: %s", adminKey)
	errorChan := make(chan error, 1)
	go func() {
		if Config.Cert != "" && Config.Key != "" {
			if err := srv.ListenAndServeTLS(Config.Cert, Config.Key); err != nil && err != http.ErrServerClosed {
				log.Println(err)
				errorChan <- err
			}
		}
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println(err)
			errorChan <- err
		}
	}()

	time.Sleep(2 * time.Second)
	select {
	case err := <-errorChan:
		return stopChan, err
	default:
		return stopChan, nil
	}
}

func StopServer() {
	srv.Shutdown(context.Background())
	log.Println("server stopped")
	stopChan <- nil
}
