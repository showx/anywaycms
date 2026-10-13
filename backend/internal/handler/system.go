package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"anywaycms/backend/internal/middleware"
	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type settingInput struct {
	SystemName   string `json:"system_name"`
	SystemNotice string `json:"system_notice"`
	TrackVisits  *int   `json:"track_visits"`
	RetainDays   *int   `json:"retain_days"`
}

type visitInput struct {
	Path    string `json:"path"`
	Referer string `json:"referer"`
}

type dayStat struct {
	Date     string `json:"date"`
	Views    int64  `json:"views"`
	Visitors int64  `json:"visitors"`
}

type nameStat struct {
	Name  string `json:"name"`
	Views int64  `json:"views"`
}

type siteStat struct {
	SiteID uint   `json:"site_id"`
	Title  string `json:"title"`
	Views  int64  `json:"views"`
}

func (h *Handler) loadSettings() (model.Setting, error) {
	var row model.Setting
	err := h.db.Order("id asc").First(&row).Error
	return row, err
}

func (h *Handler) PublicSystem(c *gin.Context) {
	row, err := h.loadSettings()
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, gin.H{
		"system_name":   row.SystemName,
		"system_notice": row.SystemNotice,
	})
}

func (h *Handler) GetSettings(c *gin.Context) {
	row, err := h.loadSettings()
	if err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, row)
}

func (h *Handler) UpdateSettings(c *gin.Context) {
	row, err := h.loadSettings()
	if err != nil {
		writeDBErr(c, err)
		return
	}
	var in settingInput
	if !bindJSON(c, &in) {
		return
	}
	name, msg := requiredText(in.SystemName, 40, "系统名称")
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	notice, msg := optionalText(in.SystemNotice, 80, "系统说明")
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	track := row.TrackVisits
	if in.TrackVisits != nil {
		if *in.TrackVisits != 0 && *in.TrackVisits != 1 {
			fail(c, http.StatusBadRequest, "访问统计开关不正确")
			return
		}
		track = *in.TrackVisits
	}
	retain := row.RetainDays
	if in.RetainDays != nil {
		if *in.RetainDays < 7 || *in.RetainDays > 365 {
			fail(c, http.StatusBadRequest, "保留天数需要在 7 到 365 之间")
			return
		}
		retain = *in.RetainDays
	}
	row.SystemName = name
	row.SystemNotice = notice
	row.TrackVisits = track
	row.RetainDays = retain
	if err := h.db.Save(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if err := h.db.Model(&row).Updates(map[string]any{
		"track_visits": track,
		"retain_days":  retain,
	}).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, row)
}

func (h *Handler) RecordVisit(c *gin.Context) {
	site, ok := currentSite(c)
	if !ok {
		return
	}
	settings, err := h.loadSettings()
	if err != nil {
		writeDBErr(c, err)
		return
	}
	if settings.TrackVisits != 1 {
		success(c, gin.H{"recorded": false})
		return
	}
	var in visitInput
	if !bindJSON(c, &in) {
		return
	}
	pagePath, msg := cleanVisitPath(in.Path)
	if msg != "" {
		fail(c, http.StatusBadRequest, msg)
		return
	}
	referer := strings.TrimSpace(in.Referer)
	if len([]rune(referer)) > 300 {
		referer = string([]rune(referer)[:300])
	}
	host, _ := c.Get(middleware.CtxHost)
	hostName, _ := host.(string)
	if settings.RetainDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -settings.RetainDays)
		h.db.Where("created_at < ?", cutoff).Delete(&model.Visit{})
	}
	row := model.Visit{
		SiteID:  site.ID,
		Host:    hostName,
		Path:    pagePath,
		Referer: referer,
		Visitor: visitorKey(c.ClientIP()),
	}
	if err := h.db.Create(&row).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, gin.H{"recorded": true})
}

func (h *Handler) Stats(c *gin.Context) {
	days := 7
	if raw := strings.TrimSpace(c.Query("days")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || (n != 7 && n != 30 && n != 90) {
			fail(c, http.StatusBadRequest, "统计范围只能是 7、30 或 90 天")
			return
		}
		days = n
	}
	var siteID uint
	if raw := strings.TrimSpace(c.Query("site_id")); raw != "" && raw != "0" {
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || n == 0 {
			fail(c, http.StatusBadRequest, "站点不正确")
			return
		}
		siteID = uint(n)
		if !h.siteExists(c, siteID) {
			return
		}
	}
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(days - 1))
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	views := h.countVisits(start, siteID, false)
	visitors := h.countVisits(start, siteID, true)
	todayViews := h.countVisits(today, siteID, false)
	todayVisitors := h.countVisits(today, siteID, true)
	daily := h.dailyStats(start, days, siteID)
	paths := h.groupedStats("path", start, siteID, 10)
	hosts := h.groupedStats("host", start, siteID, 10)
	sites := h.siteStats(start, siteID)

	success(c, gin.H{
		"days":    days,
		"site_id": siteID,
		"summary": gin.H{
			"views":          views,
			"visitors":       visitors,
			"today_views":    todayViews,
			"today_visitors": todayVisitors,
		},
		"daily": daily,
		"paths": paths,
		"hosts": hosts,
		"sites": sites,
	})
}

func (h *Handler) countVisits(since time.Time, siteID uint, distinct bool) int64 {
	tx := h.db.Model(&model.Visit{}).Where("created_at >= ?", since)
	if siteID > 0 {
		tx = tx.Where("site_id = ?", siteID)
	}
	if distinct {
		var row struct {
			N int64 `gorm:"column:n"`
		}
		tx.Select("count(distinct visitor) as n").Scan(&row)
		return row.N
	}
	var n int64
	tx.Count(&n)
	return n
}

func (h *Handler) dailyStats(start time.Time, days int, siteID uint) []dayStat {
	type row struct {
		Label    string
		Views    int64
		Visitors int64
	}
	tx := h.db.Model(&model.Visit{}).Where("created_at >= ?", start)
	if siteID > 0 {
		tx = tx.Where("site_id = ?", siteID)
	}
	var rows []row
	tx.Select("strftime('%Y-%m-%d', created_at) as label, count(*) as views, count(distinct visitor) as visitors").
		Group("label").Scan(&rows)
	found := map[string]row{}
	for _, item := range rows {
		found[item.Label] = item
	}
	out := make([]dayStat, 0, days)
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		item := found[day]
		out = append(out, dayStat{Date: day, Views: item.Views, Visitors: item.Visitors})
	}
	return out
}

func (h *Handler) groupedStats(column string, start time.Time, siteID uint, limit int) []nameStat {
	if column != "path" && column != "host" {
		return []nameStat{}
	}
	type row struct {
		Label string
		Views int64
	}
	tx := h.db.Model(&model.Visit{}).Where("created_at >= ?", start)
	if siteID > 0 {
		tx = tx.Where("site_id = ?", siteID)
	}
	var rows []row
	tx.Select(column + " as label, count(*) as views").Group("label").Order("views desc, label asc").Limit(limit).Scan(&rows)
	out := make([]nameStat, 0, len(rows))
	for _, item := range rows {
		if item.Label == "" {
			continue
		}
		out = append(out, nameStat{Name: item.Label, Views: item.Views})
	}
	return out
}

func (h *Handler) siteStats(start time.Time, siteID uint) []siteStat {
	type row struct {
		SiteID uint
		Views  int64
	}
	tx := h.db.Model(&model.Visit{}).Where("created_at >= ?", start)
	if siteID > 0 {
		tx = tx.Where("site_id = ?", siteID)
	}
	var rows []row
	tx.Select("site_id, count(*) as views").Group("site_id").Order("views desc").Scan(&rows)
	out := make([]siteStat, 0, len(rows))
	for _, item := range rows {
		var site model.Site
		title := "已删除站点"
		if err := h.db.Select("id", "title").First(&site, item.SiteID).Error; err == nil {
			title = site.Title
		}
		out = append(out, siteStat{SiteID: item.SiteID, Title: title, Views: item.Views})
	}
	return out
}

func cleanVisitPath(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "/"
	}
	if !strings.HasPrefix(raw, "/") || strings.Contains(raw, "..") || strings.ContainsAny(raw, "?#\\") {
		return "", "页面路径不正确"
	}
	if strings.HasPrefix(raw, "/admin") || strings.HasPrefix(raw, "/api") {
		return "", "这个路径不计入访问"
	}
	if len(raw) > 300 {
		return "", "路径过长"
	}
	return raw, ""
}

func visitorKey(ip string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "unknown"
	}
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:8])
}
