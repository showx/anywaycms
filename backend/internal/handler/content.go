package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type contentInput struct {
	CategoryID     uint   `json:"category_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Slug           string `json:"slug"`
	Summary        string `json:"summary"`
	Body           string `json:"body"`
	Cover          string `json:"cover"`
	Status         string `json:"status"`
	SEOTitle       string `json:"seo_title"`
	SEOKeywords    string `json:"seo_keywords"`
	SEODescription string `json:"seo_description"`
	Sort           int    `json:"sort"`
}

func (h *Handler) categoryMap(siteID uint) (map[uint]model.Category, error) {
	var rows []model.Category
	if err := h.db.Where("site_id = ?", siteID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]model.Category, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

func validateContent(in contentInput) (model.Content, string) {
	title, msg := requiredText(in.Title, 120, "标题")
	if msg != "" {
		return model.Content{}, msg
	}
	summary, msg := optionalText(in.Summary, 300, "摘要")
	if msg != "" {
		return model.Content{}, msg
	}
	cover, msg := optionalText(in.Cover, 300, "封面地址")
	if msg != "" {
		return model.Content{}, msg
	}
	seoTitle, msg := optionalText(in.SEOTitle, 120, "SEO 标题")
	if msg != "" {
		return model.Content{}, msg
	}
	seoKeywords, msg := optionalText(in.SEOKeywords, 150, "SEO 关键词")
	if msg != "" {
		return model.Content{}, msg
	}
	seoDescription, msg := optionalText(in.SEODescription, 300, "SEO 描述")
	if msg != "" {
		return model.Content{}, msg
	}
	typ := strings.TrimSpace(in.Type)
	if typ != "article" && typ != "page" {
		return model.Content{}, "类型只能是文章或单页"
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		status = "draft"
	}
	if status != "draft" && status != "published" {
		return model.Content{}, "状态不正确"
	}
	slug, ok := normalizeSlug(in.Slug)
	if !ok {
		return model.Content{}, "别名不能包含空格或 / ? # 等符号"
	}
	body := strings.TrimSpace(in.Body)
	if len(body) > 500000 {
		return model.Content{}, "正文过长"
	}
	categoryID := in.CategoryID
	if typ == "page" {
		categoryID = 0
	}
	if typ == "article" && categoryID == 0 {
		return model.Content{}, "文章需要选择栏目"
	}
	return model.Content{
		CategoryID:     categoryID,
		Type:           typ,
		Title:          title,
		Slug:           slug,
		Summary:        summary,
		Body:           body,
		Cover:          cover,
		Status:         status,
		SEOTitle:       seoTitle,
		SEOKeywords:    seoKeywords,
		SEODescription: seoDescription,
		Sort:           in.Sort,
	}, ""
}

func (h *Handler) ListContents(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	page, size := pageParams(c)
	typ := strings.TrimSpace(c.Query("type"))
	if typ != "" && typ != "article" && typ != "page" {
		fail(c, http.StatusBadRequest, "类型不正确")
		return
	}
	q := strings.NewReplacer("%", "", "_", "").Replace(strings.TrimSpace(c.Query("q")))
	var categoryID uint
	filterCategory := strings.TrimSpace(c.Query("category_id")) != ""
	if filterCategory {
		parsed, err := parseUintQuery(c.Query("category_id"))
		if err != nil {
			fail(c, http.StatusBadRequest, "栏目不正确")
			return
		}
		categoryID = parsed
	}
	build := func() *gorm.DB {
		tx := h.db.Model(&model.Content{}).Where("site_id = ?", siteID)
		if typ != "" {
			tx = tx.Where("type = ?", typ)
		}
		if filterCategory {
			tx = tx.Where("category_id = ?", categoryID)
		}
		if q != "" {
			tx = tx.Where("title LIKE ?", "%"+q+"%")
		}
		return tx
	}
	var total int64
	if err := build().Count(&total).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var rows []model.Content
	if err := build().Order("updated_at desc, id desc").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	cats, err := h.categoryMap(siteID)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	list := make([]ContentBrief, 0, len(rows))
	for _, row := range rows {
		list = append(list, toBrief(row, cats))
	}
	success(c, gin.H{"list": list, "total": total, "page": page, "page_size": size})
}

func (h *Handler) GetContent(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var item model.Content
	if err := h.db.First(&item, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, item)
}

func (h *Handler) CreateContent(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	var in contentInput
	if !bindJSON(c, &in) {
		return
	}
	item, msg := validateContent(in)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	if item.Type == "article" && !h.categoryBelongs(c, siteID, item.CategoryID) {
		return
	}
	auto := item.Slug == ""
	if auto {
		item.Slug = fmt.Sprintf("tmp%d", time.Now().UnixNano())
	}
	taken, err := h.slugTaken(siteID, item.Slug, item.Type, 0)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	if taken {
		fail(c, http.StatusBadRequest, "别名已存在")
		return
	}
	item.SiteID = siteID
	if item.Status == "published" {
		now := time.Now()
		item.PublishedAt = &now
	}
	if err := h.db.Create(&item).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if item.CategoryID == 0 {
		if err := h.db.Model(&item).Update("category_id", 0).Error; err != nil {
			writeDBErr(c, err)
			return
		}
	}
	if auto {
		item.Slug = fmt.Sprintf("c%d", item.ID)
		if err := h.db.Model(&item).Update("slug", item.Slug).Error; err != nil {
			writeDBErr(c, err)
			return
		}
	}
	success(c, item)
}

func (h *Handler) UpdateContent(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var item model.Content
	if err := h.db.First(&item, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var in contentInput
	if !bindJSON(c, &in) {
		return
	}
	next, msg := validateContent(in)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	if next.Slug == "" {
		next.Slug = item.Slug
	}
	if next.Type == "article" && !h.categoryBelongs(c, item.SiteID, next.CategoryID) {
		return
	}
	taken, err := h.slugTaken(item.SiteID, next.Slug, next.Type, item.ID)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	if taken {
		fail(c, http.StatusBadRequest, "别名已存在")
		return
	}
	if next.Status == "published" && item.PublishedAt == nil {
		now := time.Now()
		item.PublishedAt = &now
	}
	item.CategoryID = next.CategoryID
	item.Type = next.Type
	item.Title = next.Title
	item.Slug = next.Slug
	item.Summary = next.Summary
	item.Body = next.Body
	item.Cover = next.Cover
	item.Status = next.Status
	item.SEOTitle = next.SEOTitle
	item.SEOKeywords = next.SEOKeywords
	item.SEODescription = next.SEODescription
	item.Sort = next.Sort
	if err := h.db.Save(&item).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if item.CategoryID == 0 {
		if err := h.db.Model(&item).Update("category_id", 0).Error; err != nil {
			writeDBErr(c, err)
			return
		}
	}
	success(c, item)
}

func (h *Handler) DeleteContent(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var item model.Content
	if err := h.db.First(&item, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ? AND link_type = ? AND link_id = ?", item.SiteID, item.Type, item.ID).Delete(&model.Menu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&item).Error
	})
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}

func (h *Handler) categoryBelongs(c *gin.Context, siteID, categoryID uint) bool {
	var count int64
	if err := h.db.Model(&model.Category{}).Where("id = ? AND site_id = ?", categoryID, siteID).Count(&count).Error; err != nil {
		writeDBErr(c, err)
		return false
	}
	if count == 0 {
		fail(c, http.StatusBadRequest, "栏目不存在")
		return false
	}
	return true
}

func parseUintQuery(raw string) (uint, error) {
	var n uint64
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0, fmt.Errorf("nan")
		}
		n = n*10 + uint64(ch-'0')
	}
	return uint(n), nil
}
