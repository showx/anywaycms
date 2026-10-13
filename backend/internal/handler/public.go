package handler

import (
	"errors"
	"net/http"

	"anywaycms/backend/internal/middleware"
	"anywaycms/backend/internal/model"
	"anywaycms/backend/internal/urlpath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) Bootstrap(c *gin.Context) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	host, _ := c.Get(middleware.CtxHost)
	var domains []model.Domain
	if err := h.db.Where("site_id = ?", site.ID).Order("is_primary desc, id asc").Find(&domains).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if domains == nil {
		domains = []model.Domain{}
	}
	site.Domains = domains

	categories := make([]model.Category, 0)
	if err := h.db.Where("site_id = ?", site.ID).Order("sort asc, id asc").Find(&categories).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var menus []model.Menu
	if err := h.db.Where("site_id = ?", site.ID).Order("sort asc, id asc").Find(&menus).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	publicMenus := make([]PublicMenu, 0, len(menus))
	for _, menu := range menus {
		url := h.menuURL(menu, site.URLMode)
		if url == "" {
			continue
		}
		publicMenus = append(publicMenus, PublicMenu{ID: menu.ID, Name: menu.Name, URL: url})
	}
	success(c, gin.H{
		"host":       host,
		"site":       site,
		"menus":      publicMenus,
		"categories": categories,
	})
}

func (h *Handler) Home(c *gin.Context) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	cats, err := h.categoryMap(site.ID)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	var latestRows []model.Content
	if err := h.db.Where("site_id = ? AND type = ? AND status = ?", site.ID, "article", "published").
		Order("published_at desc, id desc").Limit(6).Find(&latestRows).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	latest := make([]ContentBrief, 0, len(latestRows))
	for _, row := range latestRows {
		latest = append(latest, toBrief(row, cats))
	}

	var categories []model.Category
	if err := h.db.Where("site_id = ?", site.ID).Order("sort asc, id asc").Find(&categories).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	blocks := make([]HomeBlock, 0, len(categories))
	for _, cat := range categories {
		var articles []model.Content
		if err := h.db.Where("site_id = ? AND category_id = ? AND type = ? AND status = ?", site.ID, cat.ID, "article", "published").
			Order("published_at desc, id desc").Limit(4).Find(&articles).Error; err != nil {
			writeDBErr(c, err)
			return
		}
		briefs := make([]ContentBrief, 0, len(articles))
		one := map[uint]model.Category{cat.ID: cat}
		for _, article := range articles {
			briefs = append(briefs, toBrief(article, one))
		}
		blocks = append(blocks, HomeBlock{Category: cat, Articles: briefs})
	}

	var pageRows []model.Content
	if err := h.db.Where("site_id = ? AND type = ? AND status = ?", site.ID, "page", "published").
		Order("sort asc, id asc").Find(&pageRows).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	pages := make([]ContentBrief, 0, len(pageRows))
	for _, row := range pageRows {
		pages = append(pages, toBrief(row, cats))
	}
	success(c, gin.H{"latest": latest, "blocks": blocks, "pages": pages})
}

func (h *Handler) PublicCategory(c *gin.Context) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	var cat model.Category
	err := h.db.Where("site_id = ? AND slug = ?", site.ID, c.Param("slug")).First(&cat).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, "栏目不存在")
		return
	}
	if err != nil {
		writeDBErr(c, err)
		return
	}
	page, size := pageParams(c)
	build := func() *gorm.DB {
		return h.db.Model(&model.Content{}).Where("site_id = ? AND category_id = ? AND type = ? AND status = ?", site.ID, cat.ID, "article", "published")
	}
	var total int64
	if err := build().Count(&total).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var rows []model.Content
	if err := build().Order("published_at desc, id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	one := map[uint]model.Category{cat.ID: cat}
	list := make([]ContentBrief, 0, len(rows))
	for _, row := range rows {
		list = append(list, toBrief(row, one))
	}
	success(c, gin.H{"category": cat, "list": list, "total": total, "page": page, "page_size": size})
}

func (h *Handler) PublicArticle(c *gin.Context) {
	h.publicContent(c, "article")
}

func (h *Handler) PublicPage(c *gin.Context) {
	h.publicContent(c, "page")
}

func (h *Handler) publicContent(c *gin.Context, typ string) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	var item model.Content
	err := h.db.Where("site_id = ? AND type = ? AND slug = ? AND status = ?", site.ID, typ, c.Param("slug"), "published").First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusNotFound, "内容不存在或尚未发布")
		return
	}
	if err != nil {
		writeDBErr(c, err)
		return
	}
	cats, err := h.categoryMap(site.ID)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, ContentDetail{
		ContentBrief:   toBrief(item, cats),
		Body:           item.Body,
		SEOTitle:       item.SEOTitle,
		SEOKeywords:    item.SEOKeywords,
		SEODescription: item.SEODescription,
	})
}

func (h *Handler) menuURL(menu model.Menu, mode string) string {
	switch menu.LinkType {
	case "home":
		return "/"
	case "url":
		if menu.URL == "" {
			return ""
		}
		return menu.URL
	case "category":
		var cat model.Category
		if err := h.db.Where("id = ? AND site_id = ?", menu.LinkID, menu.SiteID).First(&cat).Error; err != nil {
			return ""
		}
		return urlpath.Join(mode, "category", cat.Slug)
	case "page", "article":
		var item model.Content
		if err := h.db.Where("id = ? AND site_id = ? AND type = ? AND status = ?", menu.LinkID, menu.SiteID, menu.LinkType, "published").First(&item).Error; err != nil {
			return ""
		}
		return urlpath.Join(mode, menu.LinkType, item.Slug)
	default:
		return ""
	}
}
