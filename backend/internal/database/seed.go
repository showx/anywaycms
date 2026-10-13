package database

import (
	"fmt"
	"html"
	"strings"
	"time"

	"anywaycms/backend/internal/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type siteSeed struct {
	Name        string
	Title       string
	Subtitle    string
	Description string
	Keywords    string
	Phone       string
	Address     string
	ICP         string
	Theme       string
	Hosts       []string
	Categories  []catSeed
	Pages       []textSeed
	Articles    []articleSeed
	Menus       []menuSeed
}

type catSeed struct {
	Name        string
	Slug        string
	Description string
}

type textSeed struct {
	Title   string
	Slug    string
	Summary string
	Body    []string
}

type articleSeed struct {
	CatSlug string
	Title   string
	Slug    string
	Summary string
	Body    []string
}

type menuSeed struct {
	Name     string
	LinkType string
	Ref      string
}

func Seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if err := tx.Create(&model.User{Username: "admin", Password: string(hash)}).Error; err != nil {
			return err
		}
		if err := applySeed(tx, starlanSeed()); err != nil {
			return err
		}
		return applySeed(tx, qingheSeed())
	})
}

func applySeed(tx *gorm.DB, in siteSeed) error {
	site := model.Site{
		Name:        in.Name,
		Title:       in.Title,
		Subtitle:    in.Subtitle,
		Description: in.Description,
		Keywords:    in.Keywords,
		Phone:       in.Phone,
		Address:     in.Address,
		ICP:         in.ICP,
		Theme:       in.Theme,
		Status:      1,
	}
	if err := tx.Create(&site).Error; err != nil {
		return err
	}
	if err := tx.Model(&site).Update("status", 1).Error; err != nil {
		return err
	}
	for i, host := range in.Hosts {
		domain := model.Domain{SiteID: site.ID, Host: host, IsPrimary: i == 0}
		if err := tx.Create(&domain).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain).Update("is_primary", i == 0).Error; err != nil {
			return err
		}
	}

	catIDs := map[string]uint{}
	for i, cat := range in.Categories {
		row := model.Category{
			SiteID:      site.ID,
			Name:        cat.Name,
			Slug:        cat.Slug,
			Description: cat.Description,
			Sort:        (i + 1) * 10,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		catIDs[cat.Slug] = row.ID
	}

	now := time.Now()
	pageIDs := map[string]uint{}
	for i, page := range in.Pages {
		item := model.Content{
			SiteID:      site.ID,
			Type:        "page",
			Title:       page.Title,
			Slug:        page.Slug,
			Summary:     page.Summary,
			Body:        paragraphs(page.Body...),
			Status:      "published",
			Sort:        (i + 1) * 10,
			PublishedAt: &now,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if err := tx.Model(&item).Update("category_id", 0).Error; err != nil {
			return err
		}
		pageIDs[page.Slug] = item.ID
	}

	articleIDs := map[string]uint{}
	for i, article := range in.Articles {
		catID := catIDs[article.CatSlug]
		if catID == 0 {
			return fmt.Errorf("栏目不存在: %s", article.CatSlug)
		}
		published := now.Add(-time.Duration(i) * time.Hour)
		item := model.Content{
			SiteID:      site.ID,
			CategoryID:  catID,
			Type:        "article",
			Title:       article.Title,
			Slug:        article.Slug,
			Summary:     article.Summary,
			Body:        paragraphs(article.Body...),
			Status:      "published",
			Sort:        (i + 1) * 10,
			PublishedAt: &published,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		articleIDs[article.Slug] = item.ID
	}

	for i, menu := range in.Menus {
		row := model.Menu{
			SiteID:   site.ID,
			Name:     menu.Name,
			LinkType: menu.LinkType,
			Sort:     (i + 1) * 10,
		}
		switch menu.LinkType {
		case "home":
		case "url":
			row.URL = menu.Ref
		case "category":
			row.LinkID = catIDs[menu.Ref]
		case "page":
			row.LinkID = pageIDs[menu.Ref]
		case "article":
			row.LinkID = articleIDs[menu.Ref]
		default:
			return fmt.Errorf("未知菜单类型: %s", menu.LinkType)
		}
		if menu.LinkType != "home" && menu.LinkType != "url" && row.LinkID == 0 {
			return fmt.Errorf("菜单目标不存在: %s %s", menu.LinkType, menu.Ref)
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func paragraphs(lines ...string) string {
	var b strings.Builder
	for _, line := range lines {
		b.WriteString("<p>")
		b.WriteString(html.EscapeString(line))
		b.WriteString("</p>")
	}
	return b.String()
}

func starlanSeed() siteSeed {
	return siteSeed{
		Name:        "星澜科技",
		Title:       "星澜科技",
		Subtitle:    "工业互联与企业数字化",
		Description: "星澜科技帮助制造和零售企业把订单、库存和生产连成一条可追踪的数据链路。",
		Keywords:    "星澜科技,MES,企业数字化",
		Phone:       "400-800-1001",
		Address:     "杭州市滨江区网商路 599 号",
		ICP:         "浙ICP备20260001号",
		Theme:       `{"layout":"corporate","primary":"#0f6e6b"}`,
		Hosts:       []string{"localhost", "127.0.0.1", "starlan.local", "www.starlan.local"},
		Categories: []catSeed{
			{Name: "公司新闻", Slug: "news", Description: "产品发布、客户实践与公司动态。"},
			{Name: "解决方案", Slug: "solution", Description: "面向工厂和门店的落地办法。"},
		},
		Pages: []textSeed{
			{
				Title:   "关于星澜",
				Slug:    "about",
				Summary: "一队从车间和门店项目里长出来的实施团队。",
				Body: []string{
					"星澜科技成立于杭州，团队来自制造执行系统和零售中台项目。我们把车间、仓库和门店的数据放进同一条链路，让计划、生产和交付可以互相看见。",
					"公司现有咨询、实施和长期运维三组人。项目从现场盘点开始，不先堆功能清单。",
				},
			},
			{
				Title:   "联系我们",
				Slug:    "contact",
				Summary: "工作日可以电话或到滨江办公室沟通。",
				Body: []string{
					"商务合作请致电 400-800-1001，或到杭州市滨江区网商路 599 号。工作日 9:00 到 18:00 有人值守。",
					"如果已经有旧系统，建议先约一次两小时的数据流诊断，再决定要不要替换。",
				},
			},
		},
		Articles: []articleSeed{
			{
				CatSlug: "news",
				Title:   "杭州研发中心启用",
				Slug:    "office-open",
				Summary: "测试环境和客户沙箱分开部署，对外支持窗口不变。",
				Body: []string{
					"杭州研发中心本月启用，测试环境与客户沙箱分开部署。新的会议室按车间早会的方式布置，方便把现场问题原样带回讨论。",
					"中心首批入驻的是 MES 产品组和数据集成组，对外支持窗口不变。",
				},
			},
			{
				CatSlug: "solution",
				Title:   "离散制造 MES 方案发布",
				Slug:    "mes-release",
				Summary: "覆盖工单、报工、质量拦截和完工入库，先跑通一条产线。",
				Body: []string{
					"这套方案面向离散装配车间，覆盖工单下发、工序报工、质量拦截和完工入库。它不替代财务系统，只把生产过程补齐。",
					"试点一般先选一条产量稳定的产线，跑通四到六周，再复制到其他车间。",
				},
			},
			{
				CatSlug: "solution",
				Title:   "连锁零售中台实践",
				Slug:    "retail-case",
				Summary: "要货、波次和缺货预警接到同一套单据模型里。",
				Body: []string{
					"一家区域连锁把门店要货、仓库波次和缺货预警接到了同一套中台。店长不再用表格汇总，周转天数在两个月里下降了一成。",
					"星澜在项目里负责单据模型和门店培训，不改动对方已经在用的收银软件。",
				},
			},
		},
		Menus: []menuSeed{
			{Name: "首页", LinkType: "home"},
			{Name: "公司新闻", LinkType: "category", Ref: "news"},
			{Name: "解决方案", LinkType: "category", Ref: "solution"},
			{Name: "关于我们", LinkType: "page", Ref: "about"},
			{Name: "联系我们", LinkType: "page", Ref: "contact"},
		},
	}
}

func qingheSeed() siteSeed {
	return siteSeed{
		Name:        "青禾文旅",
		Title:       "青禾文旅",
		Subtitle:    "山水之间的在地旅行",
		Description: "青禾文旅做西湖周边的在地行走、茶季体验和夜间导览，把一座城的日常安排成可报名的行程。",
		Keywords:    "青禾文旅,西湖,茶季,徒步",
		Phone:       "0571-8800-2211",
		Address:     "杭州市西湖区龙井路 1 号",
		ICP:         "浙ICP备20260018号",
		Theme:       `{"layout":"magazine","primary":"#9a3412"}`,
		Hosts:       []string{"qinghe.local", "www.qinghe.local"},
		Categories: []catSeed{
			{Name: "旅行手记", Slug: "notes", Description: "走过的路线、体力和天气说明。"},
			{Name: "当季活动", Slug: "events", Description: "正在报名的茶季与夜游。"},
		},
		Pages: []textSeed{
			{
				Title:   "关于青禾",
				Slug:    "about",
				Summary: "一小队本地向导，控制人数，出发前确认名单。",
				Body: []string{
					"青禾文旅是一小队本地向导，长期走西湖山区、茶园和老城区。我们把一条路线的集合时间、体力要求和备用雨天方案写清楚，再开放报名。",
					"团队不做打卡团，每场控制人数，向导会在出发前一天确认名单。",
				},
			},
			{
				Title:   "联系我们",
				Slug:    "contact",
				Summary: "茶季周末建议提前预约。",
				Body: []string{
					"报名和合作请致电 0571-8800-2211。工作室在杭州市西湖区龙井路 1 号，茶季周末建议提前预约。",
					"如果是学校或公司的小团，请至少提前一周说明人数和是否需要素食。",
				},
			},
		},
		Articles: []articleSeed{
			{
				CatSlug: "notes",
				Title:   "龙井清晨徒步",
				Slug:    "longjing-walk",
				Summary: "从茶田步道走回公路，晴天大约两小时。",
				Body: []string{
					"这条线从龙井村口出发，经过茶田步道再回到公路。晴天大约两小时，其中有一段湿滑石阶。",
					"集合时间是清晨七点。向导会准备一份当季的茶园说明，不包含早餐。",
				},
			},
			{
				CatSlug: "events",
				Title:   "春茶季开放日",
				Slug:    "tea-open",
				Summary: "周六上午看摊青和炒制，名额二十人。",
				Body: []string{
					"春茶季的开放日安排在周六上午，可以看摊青和炒制，也可以只坐下来喝茶。名额二十人。",
					"若前一晚有雨，制茶演示会改到室内，行走部分取消，已报名的人会收到短信。",
				},
			},
			{
				CatSlug: "events",
				Title:   "河坊街夜游",
				Slug:    "night-walk",
				Summary: "两小时步行，讲店铺和河道的旧功能。",
				Body: []string{
					"夜游从河坊街南端走到中河桥，沿途讲的是店铺和河道的旧功能，不是传说清单。全程两小时，以步行轻松为主。",
					"小朋友可以参加。路线有两处台阶，需要无障碍通道的客人请提前告诉我们。",
				},
			},
		},
		Menus: []menuSeed{
			{Name: "首页", LinkType: "home"},
			{Name: "旅行手记", LinkType: "category", Ref: "notes"},
			{Name: "当季活动", LinkType: "category", Ref: "events"},
			{Name: "关于青禾", LinkType: "page", Ref: "about"},
			{Name: "联系我们", LinkType: "page", Ref: "contact"},
		},
	}
}
