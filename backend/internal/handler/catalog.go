package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type categoryInput struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
}

type menuInput struct {
	Name     string `json:"name"`
	LinkType string `json:"link_type"`
	LinkID   uint   `json:"link_id"`
	URL      string `json:"url"`
	Sort     int    `json:"sort"`
}

func normalizeSlug(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", true
	}
	if len([]rune(value)) > 80 || strings.ContainsAny(value, " \t/?#%&\\") {
		return "", false
	}
	return value, true
}

func (h *Handler) slugTaken(siteID uint, slug, kind string, exceptID uint) (bool, error) {
	var count int64
	var err error
	switch kind {
	case "category":
		err = h.db.Model(&model.Category{}).Where("site_id = ? AND slug = ? AND id <> ?", siteID, slug, exceptID).Count(&count).Error
	case "article", "page":
		err = h.db.Model(&model.Content{}).Where("site_id = ? AND type = ? AND slug = ? AND id <> ?", siteID, kind, slug, exceptID).Count(&count).Error
	default:
		return false, errors.New("未知别名类型")
	}
	return count > 0, err
}

func (h *Handler) ListCategories(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	var rows []model.Category
	if err := h.db.Where("site_id = ?", siteID).Order("sort asc, id asc").Find(&rows).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	out := make([]CategoryView, 0, len(rows))
	for _, row := range rows {
		var count int64
		if err := h.db.Model(&model.Content{}).Where("category_id = ?", row.ID).Count(&count).Error; err != nil {
			writeDBErr(c, err)
			return
		}
		out = append(out, CategoryView{Category: row, ContentCount: count})
	}
	success(c, out)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	var in categoryInput
	if !bindJSON(c, &in) {
		return
	}
	row, msg := h.prepareCategory(in)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	row.SiteID = siteID
	auto := row.Slug == ""
	if auto {
		row.Slug = fmt.Sprintf("tmp%d", time.Now().UnixNano())
	}
	taken, err := h.slugTaken(siteID, row.Slug, "category", 0)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	if taken {
		fail(c, http.StatusBadRequest, "栏目别名已存在")
		return
	}
	if err := h.db.Create(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if auto {
		row.Slug = fmt.Sprintf("cat%d", row.ID)
		if err := h.db.Model(&row).Update("slug", row.Slug).Error; err != nil {
			writeDBErr(c, err)
			return
		}
	}
	success(c, row)
}

func (h *Handler) UpdateCategory(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var row model.Category
	if err := h.db.First(&row, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var in categoryInput
	if !bindJSON(c, &in) {
		return
	}
	next, msg := h.prepareCategory(in)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	if next.Slug == "" {
		next.Slug = row.Slug
	}
	taken, err := h.slugTaken(row.SiteID, next.Slug, "category", row.ID)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	if taken {
		fail(c, http.StatusBadRequest, "栏目别名已存在")
		return
	}
	row.Name = next.Name
	row.Slug = next.Slug
	row.Description = next.Description
	row.Sort = next.Sort
	if err := h.db.Save(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, row)
}

func (h *Handler) prepareCategory(in categoryInput) (model.Category, string) {
	name, msg := requiredText(in.Name, 40, "栏目名称")
	if msg != "" {
		return model.Category{}, msg
	}
	desc, msg := optionalText(in.Description, 200, "栏目说明")
	if msg != "" {
		return model.Category{}, msg
	}
	slug, ok := normalizeSlug(in.Slug)
	if !ok {
		return model.Category{}, "别名不能包含空格或 / ? # 等符号"
	}
	return model.Category{Name: name, Slug: slug, Description: desc, Sort: in.Sort}, ""
}

func (h *Handler) DeleteCategory(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var row model.Category
	if err := h.db.First(&row, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var count int64
	if err := h.db.Model(&model.Content{}).Where("category_id = ?", row.ID).Count(&count).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if count > 0 {
		fail(c, http.StatusBadRequest, "栏目下还有内容，请先移走或删除")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ? AND link_type = ? AND link_id = ?", row.SiteID, "category", row.ID).Delete(&model.Menu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&row).Error
	})
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}

func (h *Handler) ListMenus(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	rows := make([]model.Menu, 0)
	if err := h.db.Where("site_id = ?", siteID).Order("sort asc, id asc").Find(&rows).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, rows)
}

func (h *Handler) CreateMenu(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	var in menuInput
	if !bindJSON(c, &in) {
		return
	}
	row, msg := h.prepareMenu(siteID, in)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	if err := h.db.Create(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, row)
}

func (h *Handler) UpdateMenu(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var row model.Menu
	if err := h.db.First(&row, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var in menuInput
	if !bindJSON(c, &in) {
		return
	}
	next, msg := h.prepareMenu(row.SiteID, in)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	row.Name = next.Name
	row.LinkType = next.LinkType
	row.LinkID = next.LinkID
	row.URL = next.URL
	row.Sort = next.Sort
	if err := h.db.Save(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, row)
}

func (h *Handler) prepareMenu(siteID uint, in menuInput) (model.Menu, string) {
	name, msg := requiredText(in.Name, 40, "菜单名称")
	if msg != "" {
		return model.Menu{}, msg
	}
	row := model.Menu{SiteID: siteID, Name: name, Sort: in.Sort}
	switch strings.TrimSpace(in.LinkType) {
	case "home":
		row.LinkType = "home"
	case "url":
		url := strings.TrimSpace(in.URL)
		if url == "" || (!strings.HasPrefix(url, "/") && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://")) {
			return model.Menu{}, "外链需要以 / 、http:// 或 https:// 开头"
		}
		if len(url) > 300 {
			return model.Menu{}, "链接过长"
		}
		row.LinkType = "url"
		row.URL = url
	case "category":
		var cat model.Category
		if err := h.db.Where("id = ? AND site_id = ?", in.LinkID, siteID).First(&cat).Error; err != nil {
			return model.Menu{}, "栏目不存在"
		}
		row.LinkType = "category"
		row.LinkID = cat.ID
	case "page", "article":
		var item model.Content
		if err := h.db.Where("id = ? AND site_id = ? AND type = ?", in.LinkID, siteID, in.LinkType).First(&item).Error; err != nil {
			return model.Menu{}, "内容不存在"
		}
		row.LinkType = in.LinkType
		row.LinkID = item.ID
	default:
		return model.Menu{}, "链接类型不正确"
	}
	return row, ""
}

func (h *Handler) DeleteMenu(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var row model.Menu
	if err := h.db.First(&row, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if err := h.db.Delete(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}
