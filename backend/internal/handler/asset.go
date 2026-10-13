package handler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
)

const maxAssetBytes = 8 << 20

var storedName = regexp.MustCompile(`^[a-f0-9]{32}\.(jpg|png|gif|webp|pdf)$`)

var extByMIME = map[string]string{
	"image/jpeg":      "jpg",
	"image/png":       "png",
	"image/gif":       "gif",
	"image/webp":      "webp",
	"application/pdf": "pdf",
}

type assetView struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	Mime      string    `json:"mime"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

func toAssetView(row model.Asset) assetView {
	return assetView{
		ID:        row.ID,
		Name:      row.Name,
		URL:       fmt.Sprintf("/uploads/%d/%s", row.SiteID, row.Stored),
		Mime:      row.Mime,
		Size:      row.Size,
		CreatedAt: row.CreatedAt,
	}
}

func displayName(raw string) string {
	raw = strings.ReplaceAll(raw, "\\", "/")
	raw = filepath.Base(raw)
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "." || raw == "/" {
		return "未命名"
	}
	runes := []rune(raw)
	if len(runes) > 80 {
		raw = string(runes[:80])
	}
	return raw
}

func (h *Handler) assetFile(siteID uint, stored string) (string, error) {
	if h.uploadDir == "" || !storedName.MatchString(stored) || siteID == 0 {
		return "", errors.New("附件路径不正确")
	}
	root, err := filepath.Abs(h.uploadDir)
	if err != nil {
		return "", err
	}
	dir := filepath.Join(root, strconv.FormatUint(uint64(siteID), 10))
	full := filepath.Join(dir, stored)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	if abs != dir && !strings.HasPrefix(abs, dir+string(os.PathSeparator)) {
		return "", errors.New("附件路径不正确")
	}
	return abs, nil
}

func (h *Handler) removeSiteUploads(siteID uint) {
	if h.uploadDir == "" || siteID == 0 {
		return
	}
	root, err := filepath.Abs(h.uploadDir)
	if err != nil {
		return
	}
	dir := filepath.Join(root, strconv.FormatUint(uint64(siteID), 10))
	abs, err := filepath.Abs(dir)
	if err != nil || !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return
	}
	_ = os.RemoveAll(abs)
}

func (h *Handler) ListAssets(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	var rows []model.Asset
	if err := h.db.Where("site_id = ?", siteID).Order("id desc").Limit(200).Find(&rows).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	out := make([]assetView, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAssetView(row))
	}
	success(c, out)
}

func (h *Handler) UploadAsset(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAssetBytes+4096)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, "请选择 JPG、PNG、GIF、WebP 或 PDF，且不超过 8MB")
		return
	}
	defer file.Close()
	buf, err := io.ReadAll(io.LimitReader(file, maxAssetBytes+1))
	if err != nil {
		fail(c, http.StatusBadRequest, "读取文件失败")
		return
	}
	if len(buf) == 0 {
		fail(c, http.StatusBadRequest, "文件是空的")
		return
	}
	if len(buf) > maxAssetBytes {
		fail(c, http.StatusBadRequest, "文件不能超过 8MB")
		return
	}
	mime := http.DetectContentType(buf)
	ext, allowed := extByMIME[mime]
	if !allowed {
		fail(c, http.StatusBadRequest, "只支持 JPG、PNG、GIF、WebP 图片和 PDF")
		return
	}
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		fail(c, http.StatusInternalServerError, "保存文件失败")
		return
	}
	stored := hex.EncodeToString(random[:]) + "." + ext
	full, err := h.assetFile(siteID, stored)
	if err != nil {
		fail(c, http.StatusInternalServerError, "保存文件失败")
		return
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		fail(c, http.StatusInternalServerError, "保存文件失败")
		return
	}
	if err := os.WriteFile(full, buf, 0o644); err != nil {
		fail(c, http.StatusInternalServerError, "保存文件失败")
		return
	}
	name := "未命名"
	if header != nil {
		name = displayName(header.Filename)
	}
	row := model.Asset{SiteID: siteID, Name: name, Stored: stored, Mime: mime, Size: int64(len(buf))}
	if err := h.db.Create(&row).Error; err != nil {
		_ = os.Remove(full)
		writeDBErr(c, err)
		return
	}
	success(c, toAssetView(row))
}

func (h *Handler) DeleteAsset(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var row model.Asset
	if err := h.db.First(&row, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	full, pathErr := h.assetFile(row.SiteID, row.Stored)
	if err := h.db.Delete(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if pathErr == nil {
		_ = os.Remove(full)
	}
	success(c, gin.H{"deleted": true})
}

func (h *Handler) ServeAsset(c *gin.Context) {
	siteID, err := strconv.ParseUint(c.Param("site"), 10, 64)
	if err != nil || siteID == 0 {
		c.Status(http.StatusNotFound)
		return
	}
	name := c.Param("name")
	var row model.Asset
	if err := h.db.Where("site_id = ? AND stored = ?", siteID, name).First(&row).Error; err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	full, err := h.assetFile(row.SiteID, row.Stored)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "public, max-age=604800")
	c.Header("Content-Type", row.Mime)
	c.File(full)
}
