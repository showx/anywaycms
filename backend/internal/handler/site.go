package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"anywaycms/backend/internal/hostutil"
	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var themeColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type siteInput struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Subtitle    string   `json:"subtitle"`
	Description string   `json:"description"`
	Keywords    string   `json:"keywords"`
	Phone       string   `json:"phone"`
	Address     string   `json:"address"`
	ICP         string   `json:"icp"`
	Theme       string   `json:"theme"`
	Mode        string   `json:"mode"`
	Status      *int     `json:"status"`
	Hosts       []string `json:"hosts"`
}

type domainInput struct {
	Host      string `json:"host"`
	IsPrimary bool   `json:"is_primary"`
}

func orderedDomains(db *gorm.DB) *gorm.DB {
	return db.Order("is_primary desc, id asc")
}

func normalizeTheme(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = `{"layout":"corporate","primary":"#0f6e6b"}`
	}
	var theme struct {
		Layout  string `json:"layout"`
		Primary string `json:"primary"`
	}
	if err := json.Unmarshal([]byte(raw), &theme); err != nil {
		return "", "主题格式不正确"
	}
	if theme.Layout != "magazine" {
		theme.Layout = "corporate"
	}
	if !themeColor.MatchString(theme.Primary) {
		theme.Primary = "#0f6e6b"
	}
	buf, err := json.Marshal(theme)
	if err != nil {
		return "", "主题格式不正确"
	}
	return string(buf), ""
}

func normalizeHosts(raw []string) ([]string, string) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		host := hostutil.Normalize(item)
		if host == "" {
			continue
		}
		if !hostutil.Valid(host) {
			return nil, "域名不正确：" + host
		}
		if _, ok := seen[host]; ok {
			continue
		}
		seen[host] = struct{}{}
		out = append(out, host)
	}
	return out, ""
}

func (h *Handler) fillSite(in siteInput, site *model.Site, creating bool) string {
	var msg string
	if site.Name, msg = requiredText(in.Name, 40, "站点名称"); msg != "" {
		return msg
	}
	if site.Title, msg = requiredText(in.Title, 60, "网站标题"); msg != "" {
		return msg
	}
	if site.Subtitle, msg = optionalText(in.Subtitle, 80, "副标题"); msg != "" {
		return msg
	}
	if site.Description, msg = optionalText(in.Description, 300, "简介"); msg != "" {
		return msg
	}
	if site.Keywords, msg = optionalText(in.Keywords, 150, "关键词"); msg != "" {
		return msg
	}
	if site.Phone, msg = optionalText(in.Phone, 40, "电话"); msg != "" {
		return msg
	}
	if site.Address, msg = optionalText(in.Address, 120, "地址"); msg != "" {
		return msg
	}
	if site.ICP, msg = optionalText(in.ICP, 80, "备案号"); msg != "" {
		return msg
	}
	theme, msg := normalizeTheme(in.Theme)
	if msg != "" {
		return msg
	}
	site.Theme = theme
	switch in.Mode {
	case "html", "cms":
		site.Mode = in.Mode
	case "":
		if creating || site.Mode == "" {
			site.Mode = "cms"
		}
	default:
		return "前台形式不正确"
	}
	site.Status = 1
	if in.Status != nil {
		if *in.Status != 0 && *in.Status != 1 {
			return "状态不正确"
		}
		site.Status = *in.Status
	}
	return ""
}

func (h *Handler) siteView(site model.Site) SiteView {
	var articles, pages, categories int64
	h.db.Model(&model.Content{}).Where("site_id = ? AND type = ?", site.ID, "article").Count(&articles)
	h.db.Model(&model.Content{}).Where("site_id = ? AND type = ?", site.ID, "page").Count(&pages)
	h.db.Model(&model.Category{}).Where("site_id = ?", site.ID).Count(&categories)
	if site.Domains == nil {
		var domains []model.Domain
		h.db.Where("site_id = ?", site.ID).Order("is_primary desc, id asc").Find(&domains)
		site.Domains = domains
	}
	if site.Domains == nil {
		site.Domains = []model.Domain{}
	}
	return SiteView{Site: site, ArticleCount: articles, PageCount: pages, CategoryCount: categories}
}

func (h *Handler) ListSites(c *gin.Context) {
	var sites []model.Site
	if err := h.db.Preload("Domains", orderedDomains).Order("id asc").Find(&sites).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	out := make([]SiteView, 0, len(sites))
	for _, site := range sites {
		if site.Domains == nil {
			site.Domains = []model.Domain{}
		}
		out = append(out, h.siteView(site))
	}
	success(c, out)
}

func (h *Handler) GetSite(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var site model.Site
	if err := h.db.Preload("Domains", orderedDomains).First(&site, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, h.siteView(site))
}

func (h *Handler) CreateSite(c *gin.Context) {
	var in siteInput
	if !bindJSON(c, &in) {
		return
	}
	site := model.Site{}
	if msg := h.fillSite(in, &site, true); msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	hosts, msg := normalizeHosts(in.Hosts)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	if len(hosts) == 0 {
		fail(c, http.StatusBadRequest, "请至少绑定一个域名")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&site).Error; err != nil {
			return err
		}
		if err := tx.Model(&site).Update("status", site.Status).Error; err != nil {
			return err
		}
		for i, host := range hosts {
			domain := model.Domain{SiteID: site.ID, Host: host, IsPrimary: i == 0}
			if err := tx.Create(&domain).Error; err != nil {
				return err
			}
			if err := tx.Model(&domain).Update("is_primary", i == 0).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		writeDBErr(c, err)
		return
	}
	var fresh model.Site
	if err := h.db.Preload("Domains", orderedDomains).First(&fresh, site.ID).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, h.siteView(fresh))
}

func (h *Handler) UpdateSite(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var site model.Site
	if err := h.db.First(&site, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var in siteInput
	if !bindJSON(c, &in) {
		return
	}
	if msg := h.fillSite(in, &site, false); msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	if err := h.db.Save(&site).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var fresh model.Site
	if err := h.db.Preload("Domains", orderedDomains).First(&fresh, site.ID).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, h.siteView(fresh))
}

func (h *Handler) DeleteSite(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, id) {
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ?", id).Delete(&model.Menu{}).Error; err != nil {
			return err
		}
		if err := tx.Where("site_id = ?", id).Delete(&model.Content{}).Error; err != nil {
			return err
		}
		if err := tx.Where("site_id = ?", id).Delete(&model.Category{}).Error; err != nil {
			return err
		}
		if err := tx.Where("site_id = ?", id).Delete(&model.Domain{}).Error; err != nil {
			return err
		}
		if err := tx.Where("site_id = ?", id).Delete(&model.Visit{}).Error; err != nil {
			return err
		}
		if err := tx.Where("site_id = ?", id).Delete(&model.Asset{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.Site{}, id).Error
	})
	if err != nil {
		writeDBErr(c, err)
		return
	}
	h.removeSiteUploads(id)
	h.removeHTMLSite(id)
	success(c, gin.H{"deleted": true})
}

func (h *Handler) CreateDomain(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	var in domainInput
	if !bindJSON(c, &in) {
		return
	}
	hosts, msg := normalizeHosts([]string{in.Host})
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	if len(hosts) != 1 {
		fail(c, http.StatusBadRequest, "请填写域名")
		return
	}
	var created model.Domain
	err := h.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.Domain{}).Where("site_id = ?", siteID).Count(&count).Error; err != nil {
			return err
		}
		primary := count == 0 || in.IsPrimary
		if primary {
			if err := tx.Model(&model.Domain{}).Where("site_id = ?", siteID).Update("is_primary", false).Error; err != nil {
				return err
			}
		}
		created = model.Domain{SiteID: siteID, Host: hosts[0], IsPrimary: primary}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		return tx.Model(&created).Update("is_primary", primary).Error
	})
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, created)
}

func (h *Handler) PrimaryDomain(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var domain model.Domain
	if err := h.db.First(&domain, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Domain{}).Where("site_id = ?", domain.SiteID).Update("is_primary", false).Error; err != nil {
			return err
		}
		return tx.Model(&domain).Update("is_primary", true).Error
	})
	if err != nil {
		writeDBErr(c, err)
		return
	}
	domain.IsPrimary = true
	success(c, domain)
}

func (h *Handler) DeleteDomain(c *gin.Context) {
	id, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var domain model.Domain
	if err := h.db.First(&domain, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	var count int64
	if err := h.db.Model(&model.Domain{}).Where("site_id = ?", domain.SiteID).Count(&count).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if count <= 1 {
		fail(c, http.StatusBadRequest, "至少保留一个域名")
		return
	}
	err := h.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Delete(&domain).Error; err != nil {
			return err
		}
		if !domain.IsPrimary {
			return nil
		}
		var next model.Domain
		if err := tx.Where("site_id = ?", domain.SiteID).Order("id asc").First(&next).Error; err != nil {
			return err
		}
		return tx.Model(&next).Update("is_primary", true).Error
	})
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, gin.H{"deleted": true})
}
