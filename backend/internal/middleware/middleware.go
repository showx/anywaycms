package middleware

import (
	"errors"
	"net/http"
	"strings"

	"anywaycms/backend/internal/hostutil"
	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

const (
	CtxSite     = "site"
	CtxHost     = "host"
	CtxUID      = "uid"
	CtxUsername = "username"
)

type Claims struct {
	UID      uint   `json:"uid"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Site-Host")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func APIHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Header("Cache-Control", "no-store")
		}
		c.Next()
	}
}

func ResolveSite(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		host := hostutil.FromRequest(c.Request)
		var domain model.Domain
		err := db.Where("host = ?", host).First(&domain).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "这个域名还没有绑定站点", "host": host})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "读取站点失败", "host": host})
			}
			c.Abort()
			return
		}
		var site model.Site
		if err := db.First(&site, domain.SiteID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "message": "站点不存在", "host": host})
			c.Abort()
			return
		}
		if site.Status != 1 {
			c.JSON(http.StatusForbidden, gin.H{"ok": false, "message": "站点已停用", "host": host})
			c.Abort()
			return
		}
		c.Set(CtxSite, site)
		c.Set(CtxHost, host)
		c.Next()
	}
}

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "请先登录"})
			c.Abort()
			return
		}
		var claims Claims
		token, err := jwt.ParseWithClaims(strings.TrimPrefix(header, "Bearer "), &claims, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid || claims.UID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "message": "登录已失效"})
			c.Abort()
			return
		}
		c.Set(CtxUID, claims.UID)
		c.Set(CtxUsername, claims.Username)
		c.Next()
	}
}
