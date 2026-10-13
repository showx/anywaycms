package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"anywaycms/backend/internal/hostutil"
	"anywaycms/backend/internal/model"
	"anywaycms/backend/internal/seo"
	"anywaycms/backend/internal/urlpath"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type seoInput struct {
	SEOTitle    string `json:"seo_title"`
	Description string `json:"description"`
	Keywords    string `json:"keywords"`
	Robots      string `json:"robots"`
	URLMode     string `json:"url_mode"`
}

type seoLink struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Loc   string `json:"loc"`
}

type seoGap struct {
	Kind  string `json:"kind"`
	ID    uint   `json:"id"`
	Title string `json:"title"`
	Path  string `json:"path"`
	Issue string `json:"issue"`
}

func normalizeRobots(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "noindex") {
		return "noindex"
	}
	return ""
}

func robotsMeta(site model.Site, index bool) string {
	if site.Robots == "noindex" {
		return "noindex,nofollow"
	}
	if !index {
		return "noindex,follow"
	}
	return "index,follow"
}

func absoluteImage(cover string) string {
	cover = strings.TrimSpace(cover)
	if strings.HasPrefix(cover, "https://") || strings.HasPrefix(cover, "http://") {
		return cover
	}
	return ""
}

func lastMod(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2006-01-02")
}

func (h *Handler) primaryHost(siteID uint, fallback string) string {
	var domain model.Domain
	err := h.db.Where("site_id = ? AND is_primary = ?", siteID, true).Order("id asc").First(&domain).Error
	if err == nil && strings.TrimSpace(domain.Host) != "" {
		return domain.Host
	}
	if err := h.db.Where("site_id = ?", siteID).Order("id asc").First(&domain).Error; err == nil && domain.Host != "" {
		return domain.Host
	}
	return fallback
}

func (h *Handler) originFor(r *http.Request, site model.Site) string {
	fallback := ""
	if r != nil {
		fallback = hostutil.FromRequest(r)
	}
	return seo.Origin(r, h.primaryHost(site.ID, fallback))
}

func (h *Handler) finishHead(r *http.Request, site model.Site, kind, pageTitle, description, keywords, image, urlPath string, index bool, published *time.Time) seo.Head {
	title := seo.Compose(pageTitle, site.Title)
	if kind == "home" {
		title = seo.OneLine(pageTitle)
		if title == "" {
			title = seo.OneLine(site.Title)
		}
	}
	head := seo.Head{
		Title:       title,
		Description: seo.OneLine(description),
		Keywords:    seo.OneLine(keywords),
		Robots:      robotsMeta(site, index),
		Canonical:   strings.TrimRight(h.originFor(r, site), "/") + urlPath,
		OGType:      "website",
		SiteName:    site.Title,
		Image:       absoluteImage(image),
	}
	if urlPath == "/" {
		head.Canonical = strings.TrimRight(h.originFor(r, site), "/") + "/"
	}
	if kind == "article" {
		head.OGType = "article"
	}
	if head.Description == "" {
		head.Description = seo.OneLine(site.Description)
	}
	if head.Keywords == "" {
		head.Keywords = seo.OneLine(site.Keywords)
	}
	head.JSONLD = seo.JSONLD(kind, head, published)
	return head
}

func (h *Handler) buildHead(r *http.Request, site model.Site, rawPath string) (seo.Head, error) {
	hit := urlpath.Parse(site.URLMode, rawPath)
	page := 1
	if r != nil {
		if n, err := strconv.Atoi(r.URL.Query().Get("page")); err == nil && n > 1 {
			page = n
		}
	}
	switch hit.Kind {
	case "home":
		title := site.SEOTitle
		if strings.TrimSpace(title) == "" {
			title = site.Title
		}
		return h.finishHead(r, site, "home", title, site.Description, site.Keywords, "", "/", true, nil), nil
	case "category":
		var cat model.Category
		err := h.db.Where("site_id = ? AND slug = ?", site.ID, hit.Slug).First(&cat).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return h.finishHead(r, site, "home", site.Title, site.Description, site.Keywords, "", hit.Path, false, nil), nil
			}
			return seo.Head{}, err
		}
		title := cat.SEOTitle
		if strings.TrimSpace(title) == "" {
			title = cat.Name
		}
		description := cat.SEODescription
		if strings.TrimSpace(description) == "" {
			description = cat.Description
		}
		return h.finishHead(r, site, "category", title, description, cat.SEOKeywords, "", urlpath.Join(site.URLMode, "category", cat.Slug), page == 1, nil), nil
	case "article", "page":
		return h.contentHead(r, site, hit.Slug, hit.Kind, hit.Path)
	default:
		return h.finishHead(r, site, "home", site.Title, site.Description, site.Keywords, "", hit.Path, false, nil), nil
	}
}

func (h *Handler) contentHead(r *http.Request, site model.Site, slug, typ, urlPath string) (seo.Head, error) {
	var item model.Content
	err := h.db.Where("site_id = ? AND type = ? AND slug = ? AND status = ?", site.ID, typ, slug, "published").First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return h.finishHead(r, site, "home", site.Title, site.Description, site.Keywords, "", urlPath, false, nil), nil
		}
		return seo.Head{}, err
	}
	title := item.SEOTitle
	if strings.TrimSpace(title) == "" {
		title = item.Title
	}
	description := item.SEODescription
	if strings.TrimSpace(description) == "" {
		description = item.Summary
	}
	return h.finishHead(r, site, typ, title, description, item.SEOKeywords, item.Cover, urlpath.Join(site.URLMode, typ, item.Slug), true, item.PublishedAt), nil
}

func (h *Handler) PublicSEO(c *gin.Context) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	head, err := h.buildHead(c.Request, site, c.Query("path"))
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, head)
}

func (h *Handler) Robots(c *gin.Context) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	body := seo.RobotsText(site.Robots != "noindex", h.originFor(c.Request, site)+"/sitemap.xml")
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}

func (h *Handler) Sitemap(c *gin.Context) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	entries, err := h.sitemapEntries(c.Request, site)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "application/xml; charset=utf-8", []byte(seo.SitemapXML(entries)))
}

func (h *Handler) sitemapEntries(r *http.Request, site model.Site) ([]seo.Entry, error) {
	origin := strings.TrimRight(h.originFor(r, site), "/")
	entries := []seo.Entry{{Loc: origin + "/", LastMod: lastMod(site.UpdatedAt)}}
	var categories []model.Category
	if err := h.db.Where("site_id = ?", site.ID).Order("sort asc, id asc").Find(&categories).Error; err != nil {
		return nil, err
	}
	for _, cat := range categories {
		entries = append(entries, seo.Entry{Loc: origin + urlpath.Join(site.URLMode, "category", cat.Slug), LastMod: lastMod(cat.UpdatedAt)})
	}
	var contents []model.Content
	if err := h.db.Where("site_id = ? AND status = ? AND type IN ?", site.ID, "published", []string{"article", "page"}).
		Order("type asc, id asc").Find(&contents).Error; err != nil {
		return nil, err
	}
	for _, item := range contents {
		entries = append(entries, seo.Entry{Loc: origin + urlpath.Join(site.URLMode, item.Type, item.Slug), LastMod: lastMod(item.UpdatedAt)})
	}
	return entries, nil
}

func (h *Handler) GetSEO(c *gin.Context) {
	site, ok := h.seoSite(c)
	if !ok {
		return
	}
	payload, err := h.seoPayload(c.Request, site)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, payload)
}

func (h *Handler) UpdateSEO(c *gin.Context) {
	site, ok := h.seoSite(c)
	if !ok {
		return
	}
	var in seoInput
	if !bindJSON(c, &in) {
		return
	}
	title, msg := optionalText(in.SEOTitle, 70, "首页标题")
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	description, msg := optionalText(in.Description, 300, "简介")
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	keywords, msg := optionalText(in.Keywords, 150, "关键词")
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	robots := normalizeRobots(in.Robots)
	if err := h.db.Model(&site).Updates(map[string]any{
		"seo_title":   title,
		"description": description,
		"keywords":    keywords,
		"robots":      robots,
		"url_mode":    urlpath.Stored(in.URLMode),
	}).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if err := h.db.First(&site, site.ID).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	payload, err := h.seoPayload(c.Request, site)
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, payload)
}

func (h *Handler) seoSite(c *gin.Context) (model.Site, bool) {
	id, ok := parseParam(c, "id")
	if !ok {
		return model.Site{}, false
	}
	var site model.Site
	if err := h.db.First(&site, id).Error; err != nil {
		writeDBErr(c, err)
		return model.Site{}, false
	}
	return site, true
}

func (h *Handler) seoPayload(r *http.Request, site model.Site) (gin.H, error) {
	entries, err := h.sitemapEntries(r, site)
	if err != nil {
		return nil, err
	}
	links := make([]seoLink, 0, len(entries))
	for _, entry := range entries {
		pathValue := strings.TrimPrefix(entry.Loc, strings.TrimRight(h.originFor(r, site), "/"))
		if pathValue == "" {
			pathValue = "/"
		}
		links = append(links, seoLink{Title: pathValue, Path: pathValue, Loc: entry.Loc})
	}
	gaps, err := h.seoGaps(site.ID, site.URLMode)
	if err != nil {
		return nil, err
	}
	robots := "index"
	if site.Robots == "noindex" {
		robots = "noindex"
	}
	return gin.H{
		"seo_title":    site.SEOTitle,
		"title":        site.Title,
		"description":  site.Description,
		"keywords":     site.Keywords,
		"robots":       robots,
		"url_mode":     urlpath.Label(site.URLMode),
		"primary_host": h.primaryHost(site.ID, hostutil.FromRequest(r)),
		"origin":       h.originFor(r, site),
		"urls":         links,
		"gaps":         gaps,
	}, nil
}

func (h *Handler) seoGaps(siteID uint, mode string) ([]seoGap, error) {
	gaps := make([]seoGap, 0)
	var categories []model.Category
	if err := h.db.Where("site_id = ? AND description = ? AND seo_description = ?", siteID, "", "").Order("id asc").Limit(30).Find(&categories).Error; err != nil {
		return nil, err
	}
	for _, cat := range categories {
		gaps = append(gaps, seoGap{Kind: "category", ID: cat.ID, Title: cat.Name, Path: urlpath.Join(mode, "category", cat.Slug), Issue: "缺少描述"})
	}
	var contents []model.Content
	if err := h.db.Where("site_id = ? AND status = ? AND summary = ? AND seo_description = ?", siteID, "published", "", "").Order("id asc").Limit(30).Find(&contents).Error; err != nil {
		return nil, err
	}
	for _, item := range contents {
		kind := "文章"
		if item.Type == "page" {
			kind = "单页"
		}
		gaps = append(gaps, seoGap{Kind: item.Type, ID: item.ID, Title: item.Title, Path: urlpath.Join(mode, item.Type, item.Slug), Issue: kind + "缺少描述"})
	}
	return gaps, nil
}

func (h *Handler) PublicRedirect(r *http.Request, urlPath string) string {
	host := hostutil.FromRequest(r)
	var domain model.Domain
	if err := h.db.Where("host = ?", host).First(&domain).Error; err != nil {
		return ""
	}
	var site model.Site
	if err := h.db.First(&site, domain.SiteID).Error; err != nil || site.Status != 1 {
		return ""
	}
	hit := urlpath.Parse(site.URLMode, urlPath)
	if !hit.Redirect {
		return ""
	}
	target := hit.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	return target
}

func (h *Handler) InjectPublic(r *http.Request, raw []byte, urlPath string) []byte {
	host := hostutil.FromRequest(r)
	var domain model.Domain
	if err := h.db.Where("host = ?", host).First(&domain).Error; err != nil {
		return raw
	}
	var site model.Site
	if err := h.db.First(&site, domain.SiteID).Error; err != nil || site.Status != 1 {
		return raw
	}
	head, err := h.buildHead(r, site, urlPath)
	if err != nil {
		return raw
	}
	return seo.Inject(raw, head)
}

func (h *Handler) InjectAdmin(raw []byte) []byte {
	name := "站群 CMS"
	var setting model.Setting
	if err := h.db.Order("id asc").First(&setting).Error; err == nil && strings.TrimSpace(setting.SystemName) != "" {
		name = setting.SystemName
	}
	page := seo.Inject(raw, seo.Head{
		Title:    name,
		Robots:   "noindex,nofollow",
		OGType:   "website",
		SiteName: name,
	})
	base := h.adminBase()
	if base == "" {
		return page
	}
	script := "<script>window.__ADMIN_BASE__=" + strconv.Quote(base) + ";</script>\n"
	idx := strings.LastIndex(strings.ToLower(string(page)), "</head>")
	if idx < 0 {
		return append([]byte(script), page...)
	}
	out := make([]byte, 0, len(page)+len(script))
	out = append(out, page[:idx]...)
	out = append(out, script...)
	out = append(out, page[idx:]...)
	return out
}
