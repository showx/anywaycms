package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"anywaycms/backend/internal/middleware"
	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	db     *gorm.DB
	secret string
}

func New(db *gorm.DB, secret string) *Handler {
	return &Handler{db: db, secret: secret}
}

func success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"ok": true, "data": data})
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"ok": false, "message": message})
}

func bindJSON(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		fail(c, http.StatusBadRequest, "请求格式不正确")
		return false
	}
	return true
}

func parseParam(c *gin.Context, name string) (uint, bool) {
	n, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || n == 0 {
		fail(c, http.StatusBadRequest, "编号不正确")
		return 0, false
	}
	return uint(n), true
}

func pageParams(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return page, size
}

func writeDBErr(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, "数据不存在")
		return
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "unique") {
		fail(c, http.StatusBadRequest, "域名或别名已存在")
		return
	}
	log.Printf("database: %v", err)
	fail(c, http.StatusInternalServerError, "数据保存失败")
}

func requiredText(value string, max int, label string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", "请填写" + label
	}
	if len([]rune(value)) > max {
		return "", label + "过长"
	}
	return value, ""
}

func optionalText(value string, max int, label string) (string, string) {
	value = strings.TrimSpace(value)
	if len([]rune(value)) > max {
		return "", label + "过长"
	}
	return value, ""
}

func currentUID(c *gin.Context) (uint, bool) {
	value, exists := c.Get(middleware.CtxUID)
	if !exists {
		fail(c, http.StatusUnauthorized, "请先登录")
		return 0, false
	}
	id, ok := value.(uint)
	if !ok || id == 0 {
		fail(c, http.StatusUnauthorized, "请先登录")
		return 0, false
	}
	return id, true
}

func currentSite(c *gin.Context) (model.Site, bool) {
	value, exists := c.Get(middleware.CtxSite)
	if !exists {
		fail(c, http.StatusInternalServerError, "站点上下文丢失")
		return model.Site{}, false
	}
	site, ok := value.(model.Site)
	if !ok {
		fail(c, http.StatusInternalServerError, "站点上下文丢失")
		return model.Site{}, false
	}
	return site, true
}

func (h *Handler) siteExists(c *gin.Context, id uint) bool {
	var count int64
	if err := h.db.Model(&model.Site{}).Where("id = ?", id).Count(&count).Error; err != nil {
		writeDBErr(c, err)
		return false
	}
	if count == 0 {
		fail(c, http.StatusNotFound, "站点不存在")
		return false
	}
	return true
}

func (h *Handler) Health(c *gin.Context) {
	success(c, gin.H{"status": "up"})
}
