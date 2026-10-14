package handler

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"anywaycms/backend/internal/hostutil"
	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	maxHTMLZip   = 32 << 20
	maxHTMLFile  = 8 << 20
	maxHTMLSite  = 64 << 20
	maxHTMLFiles = 2000
	maxHTMLText  = 1 << 20
)

type userError struct {
	msg string
}

func (e userError) Error() string { return e.msg }

func siteMode(mode string) string {
	if mode == "html" {
		return "html"
	}
	return "cms"
}

var allowedExt = map[string]string{
	".html":        "text/html",
	".htm":         "text/html",
	".css":         "text/css",
	".js":          "text/javascript",
	".mjs":         "text/javascript",
	".json":        "application/json",
	".txt":         "text/plain; charset=utf-8",
	".xml":         "application/xml",
	".svg":         "image/svg+xml",
	".png":         "image/png",
	".jpg":         "image/jpeg",
	".jpeg":        "image/jpeg",
	".gif":         "image/gif",
	".webp":        "image/webp",
	".avif":        "image/avif",
	".ico":         "image/x-icon",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
	".ttf":         "font/ttf",
	".otf":         "font/otf",
	".eot":         "application/vnd.ms-fontobject",
	".map":         "application/json",
	".webmanifest": "application/manifest+json",
	".pdf":         "application/pdf",
	".mp4":         "video/mp4",
	".webm":        "video/webm",
	".mp3":         "audio/mpeg",
	".wav":         "audio/wav",
}

var editableExt = map[string]struct{}{
	".html": {}, ".htm": {}, ".css": {}, ".js": {}, ".mjs": {},
	".json": {}, ".txt": {}, ".xml": {}, ".svg": {}, ".webmanifest": {},
}

var errSkipZip = errors.New("skip zip entry")

var attrRef = regexp.MustCompile(`(?i)(\s(?:href|src|action|poster)\s*=\s*)(["'])([^"']*)(["'])`)
var cssURL = regexp.MustCompile(`(?i)url\(\s*([^)]+)\)`)

type htmlFileView struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Editable bool   `json:"editable"`
}

type htmlSaveInput struct {
	Path    string  `json:"path"`
	Content *string `json:"content"`
}

func (h *Handler) htmlBase() (string, error) {
	if strings.TrimSpace(h.uploadDir) == "" {
		return "", errors.New("没有文件目录")
	}
	upload, err := filepath.Abs(h.uploadDir)
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(filepath.Dir(upload), "htmlsites"))
}

func (h *Handler) htmlDir(siteID uint) (string, error) {
	if siteID == 0 {
		return "", userError{"站点不正确"}
	}
	base, err := h.htmlBase()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, strconv.FormatUint(uint64(siteID), 10))
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if abs != base && !strings.HasPrefix(abs, base+string(os.PathSeparator)) {
		return "", userError{"路径不正确"}
	}
	return abs, nil
}

func (h *Handler) removeHTMLSite(siteID uint) {
	dir, err := h.htmlDir(siteID)
	if err != nil {
		return
	}
	base, err := h.htmlBase()
	if err != nil || !strings.HasPrefix(dir, base+string(os.PathSeparator)) {
		return
	}
	_ = os.RemoveAll(dir)
}

func cleanRel(raw string) (string, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	raw = strings.TrimPrefix(raw, "/")
	if raw == "" || strings.Contains(raw, "\x00") {
		return "", userError{"路径不正确"}
	}
	clean := path.Clean(raw)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || len(clean) > 200 {
		return "", userError{"路径不正确"}
	}
	ext := strings.ToLower(path.Ext(clean))
	if _, ok := allowedExt[ext]; !ok {
		return "", userError{"不能上传 " + path.Base(clean) + "，只支持网页、样式、脚本、图片和字体"}
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return "", userError{"路径不正确"}
		}
		if strings.ContainsAny(part, `<>:"|?*`) || len([]rune(part)) > 80 {
			return "", userError{"路径不正确"}
		}
	}
	return clean, nil
}

func editable(rel string) bool {
	_, ok := editableExt[strings.ToLower(path.Ext(rel))]
	return ok
}

func (h *Handler) absRel(siteID uint, rel string) (string, string, error) {
	rel, err := cleanRel(rel)
	if err != nil {
		return "", "", err
	}
	root, err := h.htmlDir(siteID)
	if err != nil {
		return "", "", err
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", "", err
	}
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return "", "", userError{"路径不正确"}
	}
	return root, abs, nil
}

func (h *Handler) writeRel(siteID uint, rel string, data []byte) error {
	if len(data) > maxHTMLFile {
		return userError{"单个文件不能超过 8MB"}
	}
	root, full, err := h.absRel(siteID, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, data, 0o644)
}

func (h *Handler) ListHTML(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	var site model.Site
	if err := h.db.First(&site, siteID).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	files, err := h.listHTML(siteID)
	if err != nil {
		failHTML(c, err)
		return
	}
	success(c, gin.H{"mode": siteMode(site.Mode), "files": files})
}

func (h *Handler) listHTML(siteID uint) ([]htmlFileView, error) {
	root, err := h.htmlDir(siteID)
	if err != nil {
		return nil, err
	}
	out := make([]htmlFileView, 0)
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return out, nil
	}
	err = filepath.WalkDir(root, func(full string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, full)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if _, err := cleanRel(rel); err != nil {
			return nil
		}
		stat, err := entry.Info()
		if err != nil {
			return err
		}
		out = append(out, htmlFileView{Path: rel, Size: stat.Size(), Editable: editable(rel)})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (h *Handler) UploadHTML(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxHTMLZip+8192)
	if err := c.Request.ParseMultipartForm(maxHTMLZip); err != nil {
		fail(c, http.StatusBadRequest, "上传失败，压缩包不要超过 32MB")
		return
	}
	if c.Request.MultipartForm == nil {
		fail(c, http.StatusBadRequest, "请选择文件")
		return
	}
	files := c.Request.MultipartForm.File["file"]
	if len(files) == 0 {
		fail(c, http.StatusBadRequest, "请选择文件")
		return
	}
	paths := c.Request.MultipartForm.Value["path"]
	if len(files) == 1 && isZipName(files[0].Filename) && len(paths) == 0 {
		h.uploadZip(c, siteID, files[0])
		return
	}
	h.uploadFiles(c, siteID, files, paths)
}

func isZipName(name string) bool {
	return strings.ToLower(path.Ext(strings.ReplaceAll(name, "\\", "/"))) == ".zip"
}

func (h *Handler) uploadZip(c *gin.Context, siteID uint, header *multipart.FileHeader) {
	if header.Size > maxHTMLZip {
		fail(c, http.StatusBadRequest, "压缩包不要超过 32MB")
		return
	}
	file, err := header.Open()
	if err != nil {
		fail(c, http.StatusBadRequest, "读取压缩包失败")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxHTMLZip+1))
	if err != nil {
		fail(c, http.StatusBadRequest, "读取压缩包失败")
		return
	}
	if len(data) > maxHTMLZip {
		fail(c, http.StatusBadRequest, "压缩包不要超过 32MB")
		return
	}
	count, err := h.installZip(siteID, data)
	if err != nil {
		failHTML(c, err)
		return
	}
	hasIndex := h.hasIndex(siteID)
	success(c, gin.H{"count": count, "replaced": true, "has_index": hasIndex})
}

func (h *Handler) uploadFiles(c *gin.Context, siteID uint, files []*multipart.FileHeader, paths []string) {
	type incoming struct {
		rel  string
		data []byte
	}
	items := make([]incoming, 0, len(files))
	var total int
	for i, header := range files {
		raw := header.Filename
		if i < len(paths) && strings.TrimSpace(paths[i]) != "" {
			raw = paths[i]
		}
		rel, err := cleanRel(raw)
		if err != nil {
			failHTML(c, err)
			return
		}
		file, err := header.Open()
		if err != nil {
			fail(c, http.StatusBadRequest, "读取文件失败")
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxHTMLFile+1))
		file.Close()
		if err != nil {
			fail(c, http.StatusBadRequest, "读取文件失败")
			return
		}
		if len(data) == 0 {
			fail(c, http.StatusBadRequest, path.Base(rel)+" 是空文件")
			return
		}
		if len(data) > maxHTMLFile {
			fail(c, http.StatusBadRequest, "单个文件不能超过 8MB")
			return
		}
		total += len(data)
		if total > maxHTMLSite || len(items) >= maxHTMLFiles {
			fail(c, http.StatusBadRequest, "整站文件不要超过 64MB、2000 个")
			return
		}
		items = append(items, incoming{rel: rel, data: data})
	}
	for _, item := range items {
		if err := h.writeRel(siteID, item.rel, item.data); err != nil {
			failHTML(c, err)
			return
		}
	}
	success(c, gin.H{"count": len(items), "replaced": false, "has_index": h.hasIndex(siteID)})
}

func (h *Handler) hasIndex(siteID uint) bool {
	for _, name := range []string{"index.html", "index.htm"} {
		_, full, err := h.absRel(siteID, name)
		if err != nil {
			continue
		}
		info, err := os.Stat(full)
		if err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

func (h *Handler) installZip(siteID uint, data []byte) (int, error) {
	dir, err := h.htmlDir(siteID)
	if err != nil {
		return 0, err
	}
	parent := filepath.Dir(dir)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return 0, err
	}
	tmp, err := os.MkdirTemp(parent, "unpack-")
	if err != nil {
		return 0, err
	}
	defer os.RemoveAll(tmp)
	count, err := extractZip(data, tmp)
	if err != nil {
		return 0, err
	}
	if err := flattenSingleRoot(tmp); err != nil {
		return 0, err
	}
	if err := replaceDir(tmp, dir); err != nil {
		return 0, err
	}
	return count, nil
}

func extractZip(data []byte, dest string) (int, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return 0, userError{"压缩包无法打开，请上传 zip 文件"}
	}
	var total uint64
	count := 0
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || file.Mode()&os.ModeSymlink != 0 {
			continue
		}
		rel, err := zipRel(file.Name)
		if errors.Is(err, errSkipZip) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if file.UncompressedSize64 > maxHTMLFile {
			return 0, userError{"单个文件不能超过 8MB"}
		}
		total += file.UncompressedSize64
		count++
		if total > maxHTMLSite || count > maxHTMLFiles {
			return 0, userError{"整站文件不要超过 64MB、2000 个"}
		}
		full := filepath.Join(dest, filepath.FromSlash(rel))
		abs, err := filepath.Abs(full)
		if err != nil || !strings.HasPrefix(abs, dest+string(os.PathSeparator)) {
			return 0, userError{"压缩包路径不正确"}
		}
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return 0, err
		}
		src, err := file.Open()
		if err != nil {
			return 0, err
		}
		dst, err := os.OpenFile(abs, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			src.Close()
			return 0, err
		}
		written, copyErr := io.Copy(dst, io.LimitReader(src, maxHTMLFile+1))
		src.Close()
		dst.Close()
		if copyErr != nil {
			return 0, copyErr
		}
		if written > maxHTMLFile {
			return 0, userError{"单个文件不能超过 8MB"}
		}
	}
	if count == 0 {
		return 0, userError{"压缩包里没有可用的网页文件"}
	}
	return count, nil
}

func zipRel(name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "/")
	if name == "" || strings.HasSuffix(name, "/") || strings.Contains(name, "\x00") {
		return "", errSkipZip
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." {
			return "", userError{"压缩包路径不正确"}
		}
		if strings.HasPrefix(part, ".") || strings.EqualFold(part, "__MACOSX") {
			return "", errSkipZip
		}
	}
	return cleanRel(name)
}

func flattenSingleRoot(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || !entries[0].IsDir() {
		return err
	}
	sub := filepath.Join(dir, entries[0].Name())
	if _, err := os.Stat(filepath.Join(sub, "index.html")); err != nil {
		if _, err2 := os.Stat(filepath.Join(sub, "index.htm")); err2 != nil {
			return nil
		}
	}
	children, err := os.ReadDir(sub)
	if err != nil {
		return err
	}
	for _, child := range children {
		if err := os.Rename(filepath.Join(sub, child.Name()), filepath.Join(dir, child.Name())); err != nil {
			return err
		}
	}
	return os.Remove(sub)
}

func replaceDir(src, dest string) error {
	backup := dest + ".replacing"
	_ = os.RemoveAll(backup)
	had := false
	if _, err := os.Stat(dest); err == nil {
		had = true
		if err := os.Rename(dest, backup); err != nil {
			return err
		}
	}
	if err := os.Rename(src, dest); err != nil {
		if had {
			_ = os.Rename(backup, dest)
		}
		return err
	}
	_ = os.RemoveAll(backup)
	return nil
}

func (h *Handler) GetHTMLFile(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	rel, err := cleanRel(c.Query("path"))
	if err != nil {
		failHTML(c, err)
		return
	}
	if !editable(rel) {
		fail(c, http.StatusBadRequest, "这个文件不能在线编辑")
		return
	}
	_, full, err := h.absRel(siteID, rel)
	if err != nil {
		failHTML(c, err)
		return
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		fail(c, http.StatusNotFound, "文件不存在")
		return
	}
	if info.Size() > maxHTMLText {
		fail(c, http.StatusBadRequest, "文件过大，请重新上传替换")
		return
	}
	body, err := os.ReadFile(full)
	if err != nil {
		fail(c, http.StatusInternalServerError, "读取文件失败")
		return
	}
	success(c, gin.H{"path": rel, "content": string(body)})
}

func (h *Handler) SaveHTMLFile(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	var in htmlSaveInput
	if !bindJSON(c, &in) {
		return
	}
	if in.Content == nil {
		fail(c, http.StatusBadRequest, "请填写文件内容")
		return
	}
	rel, err := cleanRel(in.Path)
	if err != nil {
		failHTML(c, err)
		return
	}
	if !editable(rel) {
		fail(c, http.StatusBadRequest, "这个文件不能在线编辑")
		return
	}
	if len(*in.Content) > maxHTMLText {
		fail(c, http.StatusBadRequest, "文件内容不要超过 1MB")
		return
	}
	if err := h.writeRel(siteID, rel, []byte(*in.Content)); err != nil {
		failHTML(c, err)
		return
	}
	success(c, gin.H{"path": rel})
}

func (h *Handler) DeleteHTMLFile(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	rel, err := cleanRel(c.Query("path"))
	if err != nil {
		failHTML(c, err)
		return
	}
	_, full, err := h.absRel(siteID, rel)
	if err != nil {
		failHTML(c, err)
		return
	}
	if err := os.Remove(full); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fail(c, http.StatusNotFound, "文件不存在")
			return
		}
		fail(c, http.StatusInternalServerError, "删除文件失败")
		return
	}
	success(c, gin.H{"deleted": true})
}

func (h *Handler) ClearHTML(c *gin.Context) {
	siteID, ok := parseParam(c, "id")
	if !ok {
		return
	}
	if !h.siteExists(c, siteID) {
		return
	}
	h.removeHTMLSite(siteID)
	success(c, gin.H{"deleted": true})
}

func failHTML(c *gin.Context, err error) {
	var known userError
	if errors.As(err, &known) {
		fail(c, http.StatusBadRequest, known.msg)
		return
	}
	if err != nil {
		log.Printf("html site: %v", err)
	}
	fail(c, http.StatusInternalServerError, "保存文件失败")
}

// ServeHTMLSite 在域名属于整站 HTML 时直接返回上传的文件。
// 后台路径仍交给原来的管理界面。
func (h *Handler) ServeHTMLSite(c *gin.Context) bool {
	urlPath := path.Clean("/" + c.Request.URL.Path)
	if h.IsAdminPath(urlPath) {
		return false
	}
	site, found, err := h.siteByRequest(c.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "message": "读取站点失败"})
		return true
	}
	if !found || siteMode(site.Mode) != "html" {
		return false
	}
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		c.Header("Allow", "GET, HEAD")
		c.Status(http.StatusMethodNotAllowed)
		return true
	}
	if site.Status != 1 {
		c.Data(http.StatusForbidden, "text/html; charset=utf-8", []byte(htmlDisabledPage))
		return true
	}
	rel, ok := h.findPublicFile(site.ID, c.Request.URL.Path)
	if !ok {
		c.Data(http.StatusNotFound, "text/html; charset=utf-8", []byte(htmlMissingPage))
		return true
	}
	h.writeStored(c, site.ID, rel)
	return true
}

func (h *Handler) siteByRequest(r *http.Request) (model.Site, bool, error) {
	host := hostutil.FromRequest(r)
	var domain model.Domain
	err := h.db.Where("host = ?", host).First(&domain).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Site{}, false, nil
	}
	if err != nil {
		return model.Site{}, false, err
	}
	var site model.Site
	if err := h.db.First(&site, domain.SiteID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Site{}, false, nil
		}
		return model.Site{}, false, err
	}
	return site, true, nil
}

func (h *Handler) findPublicFile(siteID uint, rawPath string) (string, bool) {
	for _, rel := range candidateRels(rawPath) {
		_, full, err := h.absRel(siteID, rel)
		if err != nil {
			continue
		}
		info, err := os.Stat(full)
		if err == nil && !info.IsDir() {
			return rel, true
		}
	}
	return "", false
}

func candidateRels(raw string) []string {
	if raw == "" {
		raw = "/"
	}
	if !strings.HasPrefix(raw, "/") {
		raw = "/" + raw
	}
	trailing := len(raw) > 1 && strings.HasSuffix(raw, "/")
	clean := path.Clean(raw)
	if strings.Contains(raw, "\\") || strings.Contains(raw, "\x00") || strings.HasPrefix(clean, "/..") {
		return nil
	}
	if clean == "/" || clean == "." {
		return []string{"index.html", "index.htm"}
	}
	rel := strings.TrimPrefix(clean, "/")
	if !validSegments(rel) {
		return nil
	}
	if trailing {
		return []string{rel + "/index.html", rel + "/index.htm"}
	}
	ext := strings.ToLower(path.Ext(rel))
	if ext != "" {
		if _, ok := allowedExt[ext]; !ok {
			return nil
		}
		return []string{rel}
	}
	return []string{rel + ".html", rel + ".htm", rel + "/index.html", rel + "/index.htm"}
}

func validSegments(rel string) bool {
	if rel == "" || len(rel) > 200 || strings.Contains(rel, "\x00") {
		return false
	}
	clean := path.Clean(rel)
	if clean != rel || strings.HasPrefix(clean, "../") || clean == ".." {
		return false
	}
	for _, part := range strings.Split(clean, "/") {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") {
			return false
		}
		if strings.ContainsAny(part, `<>:"|?*`) {
			return false
		}
	}
	return true
}

func (h *Handler) serveExact(c *gin.Context, siteID uint, rel string) bool {
	_, full, err := h.absRel(siteID, rel)
	if err != nil {
		return false
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return false
	}
	h.writeStored(c, siteID, rel)
	return true
}

func (h *Handler) writeStored(c *gin.Context, siteID uint, rel string) {
	_, full, err := h.absRel(siteID, rel)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	c.Header("Cache-Control", "no-cache")
	c.Header("Content-Type", allowedExt[strings.ToLower(path.Ext(rel))])
	host, rewrite := previewHost(c.Request)
	ext := strings.ToLower(path.Ext(rel))
	if rewrite && (ext == ".html" || ext == ".htm" || ext == ".css") {
		body, err := os.ReadFile(full)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		c.Data(http.StatusOK, allowedExt[ext], []byte(rewritePreviewRefs(ext, string(body), host)))
		return
	}
	c.File(full)
}

func previewHost(r *http.Request) (string, bool) {
	preview := ""
	if host := strings.TrimSpace(r.Header.Get("X-Site-Host")); host != "" {
		preview = hostutil.Normalize(host)
	} else if host := strings.TrimSpace(r.URL.Query().Get("host")); host != "" {
		preview = hostutil.Normalize(host)
	}
	if preview == "" || preview == hostutil.Normalize(r.Host) {
		return "", false
	}
	return preview, true
}

func rewritePreviewRefs(ext, body, host string) string {
	if ext == ".html" || ext == ".htm" {
		body = attrRef.ReplaceAllStringFunc(body, func(match string) string {
			parts := attrRef.FindStringSubmatch(match)
			if len(parts) != 5 || parts[2] != parts[4] {
				return match
			}
			return parts[1] + parts[2] + addHostQuery(parts[3], host) + parts[4]
		})
	}
	if ext == ".html" || ext == ".htm" || ext == ".css" {
		body = cssURL.ReplaceAllStringFunc(body, func(match string) string {
			parts := cssURL.FindStringSubmatch(match)
			if len(parts) != 2 {
				return match
			}
			raw := strings.TrimSpace(parts[1])
			quote := ""
			if strings.HasPrefix(raw, `"`) || strings.HasPrefix(raw, "'") {
				quote = raw[:1]
				raw = strings.TrimSuffix(strings.TrimPrefix(raw, quote), quote)
			}
			if quote == "" {
				quote = `"`
			}
			return "url(" + quote + addHostQuery(strings.TrimSpace(raw), host) + quote + ")"
		})
	}
	return body
}

func addHostQuery(raw, host string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") || strings.HasPrefix(strings.ToLower(raw), "data:") ||
		strings.HasPrefix(strings.ToLower(raw), "mailto:") || strings.HasPrefix(strings.ToLower(raw), "tel:") ||
		strings.HasPrefix(strings.ToLower(raw), "javascript:") {
		return raw
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(raw, "//") {
		return raw
	}
	if strings.Contains(raw, "host=") {
		return raw
	}
	frag := ""
	if i := strings.Index(raw, "#"); i >= 0 {
		frag = raw[i:]
		raw = raw[:i]
	}
	escaped := url.QueryEscape(host)
	if strings.Contains(raw, "?") {
		return raw + "&host=" + escaped + frag
	}
	return raw + "?host=" + escaped + frag
}

const htmlDisabledPage = `<!DOCTYPE html><html lang="zh-CN"><meta charset="utf-8"><title>站点已停用</title><body><p>站点已停用</p></body></html>`
const htmlMissingPage = `<!DOCTYPE html><html lang="zh-CN"><meta charset="utf-8"><title>页面不存在</title><body><p>页面不存在</p></body></html>`
