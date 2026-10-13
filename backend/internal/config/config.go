package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Addr      string
	DBPath    string
	JWTSecret string
	StaticDir string
	UploadDir string
}

func Load() Config {
	root := projectRoot()
	dbPath := env("DB_PATH", filepath.Join("data", "anywaycms.db"))
	staticDir := env("STATIC_DIR", filepath.Join("frontend", "dist"))
	uploadDir := env("UPLOAD_DIR", filepath.Join("data", "uploads"))
	if !filepath.IsAbs(dbPath) {
		dbPath = filepath.Join(root, dbPath)
	}
	if !filepath.IsAbs(staticDir) {
		staticDir = filepath.Join(root, staticDir)
	}
	if !filepath.IsAbs(uploadDir) {
		uploadDir = filepath.Join(root, uploadDir)
	}
	return Config{
		Addr:      env("ADDR", ":8080"),
		DBPath:    dbPath,
		JWTSecret: env("JWT_SECRET", "anywaycms-dev-secret-change-me"),
		StaticDir: staticDir,
		UploadDir: uploadDir,
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func projectRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for i := 0; i < 6; i++ {
		if fileExists(filepath.Join(dir, "go.mod")) && dirExists(filepath.Join(dir, "frontend")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
