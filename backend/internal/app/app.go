package app

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"anywaycms/backend/internal/config"
	"anywaycms/backend/internal/database"
	"anywaycms/backend/internal/model"
	"anywaycms/backend/internal/router"

	"github.com/gin-gonic/gin"
)

func Run() error {
	cfg := config.Load()
	if cfg.JWTSecret == "anywaycms-dev-secret-change-me" {
		log.Printf("正在使用开发用 JWT 密钥，正式环境请设置环境变量 JWT_SECRET")
	}
	if os.Getenv("GIN_MODE") == "" && os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	var domains []model.Domain
	if err := db.Order("host asc").Find(&domains).Error; err != nil {
		return err
	}
	for _, domain := range domains {
		log.Printf("域名 %s -> 站点 %d", domain.Host, domain.SiteID)
	}
	if _, err := os.Stat(filepath.Join(cfg.StaticDir, "index.html")); err != nil {
		log.Printf("还没有前端构建。可在 frontend 目录执行 npm install && npm run build，或 npm run dev 后打开 http://127.0.0.1:5173")
	}

	engine := router.New(db, cfg.JWTSecret, cfg.StaticDir, cfg.UploadDir)
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("站群 CMS 已启动：%s", listenURL(cfg.Addr))
		var setting model.Setting
		adminURL := listenURL(cfg.Addr)
		if err := db.Order("id asc").First(&setting).Error; err == nil && setting.AdminPath != "" {
			adminURL += "/" + setting.AdminPath
		}
		log.Printf("后台入口：%s ，账号 admin / admin123（已有数据库不会重置密码）", adminURL)
		errCh <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func listenURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	return "http://" + addr
}
