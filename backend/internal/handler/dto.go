package handler

import (
	"time"

	"anywaycms/backend/internal/model"
)

type SiteView struct {
	model.Site
	ArticleCount  int64 `json:"article_count"`
	PageCount     int64 `json:"page_count"`
	CategoryCount int64 `json:"category_count"`
}

type CategoryView struct {
	model.Category
	ContentCount int64 `json:"content_count"`
}

type ContentBrief struct {
	ID           uint       `json:"id"`
	CategoryID   uint       `json:"category_id"`
	CategoryName string     `json:"category_name"`
	CategorySlug string     `json:"category_slug"`
	Type         string     `json:"type"`
	Title        string     `json:"title"`
	Slug         string     `json:"slug"`
	Summary      string     `json:"summary"`
	Cover        string     `json:"cover"`
	Status       string     `json:"status"`
	PublishedAt  *time.Time `json:"published_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type ContentDetail struct {
	ContentBrief
	Body           string `json:"body"`
	SEOTitle       string `json:"seo_title"`
	SEOKeywords    string `json:"seo_keywords"`
	SEODescription string `json:"seo_description"`
}

type PublicMenu struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type HomeBlock struct {
	Category model.Category `json:"category"`
	Articles []ContentBrief `json:"articles"`
}

func toBrief(item model.Content, cats map[uint]model.Category) ContentBrief {
	brief := ContentBrief{
		ID:          item.ID,
		CategoryID:  item.CategoryID,
		Type:        item.Type,
		Title:       item.Title,
		Slug:        item.Slug,
		Summary:     item.Summary,
		Cover:       item.Cover,
		Status:      item.Status,
		PublishedAt: item.PublishedAt,
		UpdatedAt:   item.UpdatedAt,
	}
	if cat, ok := cats[item.CategoryID]; ok {
		brief.CategoryName = cat.Name
		brief.CategorySlug = cat.Slug
	}
	return brief
}
