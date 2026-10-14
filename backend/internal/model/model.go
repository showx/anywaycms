package model

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64;not null" json:"username"`
	Password  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Site struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:80;not null" json:"name"`
	Title       string    `gorm:"size:120;not null" json:"title"`
	Subtitle    string    `gorm:"size:200" json:"subtitle"`
	Description string    `gorm:"size:500" json:"description"`
	Keywords    string    `gorm:"size:300" json:"keywords"`
	SEOTitle    string    `gorm:"size:200" json:"seo_title"`
	Robots      string    `gorm:"size:16" json:"robots"`
	URLMode     string    `gorm:"size:16" json:"url_mode"`
	Phone       string    `gorm:"size:40" json:"phone"`
	Address     string    `gorm:"size:200" json:"address"`
	ICP         string    `gorm:"size:80" json:"icp"`
	Theme       string    `gorm:"size:500" json:"theme"`
	Mode        string    `gorm:"size:16" json:"mode"`
	Status      int       `gorm:"not null" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Domains     []Domain  `gorm:"foreignKey:SiteID" json:"domains"`
}

type Domain struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SiteID    uint      `gorm:"index;not null" json:"site_id"`
	Host      string    `gorm:"uniqueIndex;size:255;not null" json:"host"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
}

type Category struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	SiteID         uint      `gorm:"not null;uniqueIndex:idx_cat_slug" json:"site_id"`
	Name           string    `gorm:"size:80;not null" json:"name"`
	Slug           string    `gorm:"size:80;not null;uniqueIndex:idx_cat_slug" json:"slug"`
	Description    string    `gorm:"size:300" json:"description"`
	SEOTitle       string    `gorm:"size:200" json:"seo_title"`
	SEOKeywords    string    `gorm:"size:300" json:"seo_keywords"`
	SEODescription string    `gorm:"size:500" json:"seo_description"`
	Sort           int       `json:"sort"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type Content struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	SiteID         uint       `gorm:"not null;index;uniqueIndex:idx_content_slug" json:"site_id"`
	CategoryID     uint       `gorm:"index" json:"category_id"`
	Type           string     `gorm:"size:16;not null;index;uniqueIndex:idx_content_slug" json:"type"`
	Title          string     `gorm:"size:200;not null" json:"title"`
	Slug           string     `gorm:"size:80;not null;uniqueIndex:idx_content_slug" json:"slug"`
	Summary        string     `gorm:"size:500" json:"summary"`
	Body           string     `json:"body"`
	Cover          string     `gorm:"size:300" json:"cover"`
	Status         string     `gorm:"size:16;not null;index" json:"status"`
	SEOTitle       string     `gorm:"size:200" json:"seo_title"`
	SEOKeywords    string     `gorm:"size:300" json:"seo_keywords"`
	SEODescription string     `gorm:"size:500" json:"seo_description"`
	Sort           int        `json:"sort"`
	PublishedAt    *time.Time `json:"published_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Setting struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	SystemName   string    `gorm:"size:80;not null" json:"system_name"`
	SystemNotice string    `gorm:"size:160" json:"system_notice"`
	AdminPath    string    `gorm:"size:32" json:"admin_path"`
	TrackVisits  int       `gorm:"not null" json:"track_visits"`
	RetainDays   int       `gorm:"not null" json:"retain_days"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Visit struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SiteID    uint      `gorm:"index;not null" json:"site_id"`
	Host      string    `gorm:"size:255;index" json:"host"`
	Path      string    `gorm:"size:300;index" json:"path"`
	Referer   string    `gorm:"size:300" json:"referer"`
	Visitor   string    `gorm:"size:64;index" json:"-"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type Asset struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SiteID    uint      `gorm:"index;not null" json:"site_id"`
	Name      string    `gorm:"size:200;not null" json:"name"`
	Stored    string    `gorm:"size:80;not null" json:"-"`
	Mime      string    `gorm:"size:80;not null" json:"mime"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type Menu struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	SiteID    uint      `gorm:"index;not null" json:"site_id"`
	Name      string    `gorm:"size:80;not null" json:"name"`
	LinkType  string    `gorm:"size:16;not null" json:"link_type"`
	LinkID    uint      `json:"link_id"`
	URL       string    `gorm:"size:300" json:"url"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
