# 站群 CMS

一套后台管理多个网站。访客打开哪个域名，就只看到绑在这个域名上的那一个站点。

技术栈：Go 接口、SQLite、Vue 3 前台和后台。

## 站点怎么分

```
浏览器域名
    -> 去掉端口后查 domains 表
    -> 找到 site_id
    -> 只读这个站点的栏目、文章、单页和菜单
```

- 一个站点就是一个网站，有自己的标题、栏目、文章、单页、菜单和外观。
- 一个站点可以绑多个域名。`example.com` 和 `www.example.com` 绑在一起时，打开的是同一个网站。
- 不同网站就新建不同站点，再绑各自的域名。
- 本地预览不必改 hosts。后台点「访问」，或打开 `/?host=qinghe.local`。

首次启动会写入两个演示站：

| 网站 | 域名 | 外观 |
| --- | --- | --- |
| 星澜科技 | `localhost`、`127.0.0.1`、`starlan.local`、`www.starlan.local` | 企业布局 |
| 青禾文旅 | `qinghe.local`、`www.qinghe.local` | 杂志布局 |

两个站都有别名 `about` 的单页，但正文不同。这是用来确认页面按站点隔离，而不是按别名全局共用。

## 运行

在项目根目录执行。

```powershell
go mod tidy
cd frontend
npm install
npm run build
cd ..
go run ./backend/cmd/server
```

打开 [http://127.0.0.1:8080](http://127.0.0.1:8080)。本机域名已经绑在星澜科技上。

后台地址不固定。启动时日志会打印「后台入口」，也可以在系统设置里自己改。不能使用 `admin`。环境变量 `ADMIN_PATH` 只在数据库里还没有路径时生效。

- 账号：`admin`
- 密码：`admin123`

数据库在 `data/anywaycms.db`。删除这个文件后重启，会重新生成演示数据和默认密码。已经改过的密码不会被覆盖。

开发时可以前后端分开跑：

```powershell
go run ./backend/cmd/server
```

```powershell
cd frontend
npm run dev
```

然后打开 [http://127.0.0.1:5173](http://127.0.0.1:5173)。接口会代理到 `8080`。

如果希望用真实域名访问，在 hosts 里加上：

```
127.0.0.1 starlan.local www.starlan.local qinghe.local www.qinghe.local
```

然后分别打开 `http://starlan.local:8080` 和 `http://qinghe.local:8080`。

## 环境变量

| 变量 | 默认值 | 作用 |
| --- | --- | --- |
| `ADDR` | `:8080` | 监听地址 |
| `DB_PATH` | `data/anywaycms.db` | SQLite 文件 |
| `JWT_SECRET` | 内置开发密钥 | 登录令牌签名，正式环境务必修改 |
| `STATIC_DIR` | `frontend/dist` | Vue 构建结果 |
| `UPLOAD_DIR` | `data/uploads` | 附件文件目录 |
| `APP_ENV` | 空 | 设为 `production` 时关闭 Gin 调试输出 |

## 后台能做什么

- 站点：名称、标题、联系方式、企业布局或杂志布局、启用和停用
- 域名：绑定、设主域名、解除。每个域名全局只能属于一个站点
- 栏目、文章、单页、菜单：都挂在当前选中的站点下。文章和单页正文用富文本排版
- 文章可以是草稿。前台只显示已发布内容
- 访问统计：按站点记录前台页面的浏览量和独立访客，可看近 7 天或 30 天、热门页面和域名
- SEO：每个站点单独设置首页标题、描述、关键词和是否收录。前台页面把标题、规范地址和结构化数据写进源代码。主域名提供 `/robots.txt` 和 `/sitemap.xml`，别名域名指向主域名
- 伪静态：在 SEO 里把链接样式改成「伪静态」后，栏目、文章和单页使用 `.html` 地址。旧地址会跳到新地址。每个站点可以单独开关
- 附件：按站点上传 JPG、PNG、GIF、WebP 和 PDF。文章编辑器可以直接插入图片，封面也可以用上传的地址
- 系统设置：后台名称、说明、是否记录访问、统计保留天数
- 修改后台密码

前台路径：

- `/` 首页
- `/category/栏目别名` 栏目
- `/article/文章别名` 文章
- `/page/单页别名` 单页
