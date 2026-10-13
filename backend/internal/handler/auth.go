package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"anywaycms/backend/internal/middleware"
	"anywaycms/backend/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type loginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type passwordInput struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (h *Handler) Login(c *gin.Context) {
	var in loginInput
	if !bindJSON(c, &in) {
		return
	}
	username := strings.TrimSpace(in.Username)
	var user model.User
	err := h.db.Where("username = ?", username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fail(c, http.StatusUnauthorized, "账号或密码不正确")
		return
	}
	if err != nil {
		writeDBErr(c, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Password)) != nil {
		fail(c, http.StatusUnauthorized, "账号或密码不正确")
		return
	}
	claims := middleware.Claims{
		UID:      user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Username,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(h.secret))
	if err != nil {
		fail(c, http.StatusInternalServerError, "登录失败")
		return
	}
	success(c, gin.H{"token": signed, "username": user.Username})
}

func (h *Handler) Me(c *gin.Context) {
	id, ok := currentUID(c)
	if !ok {
		return
	}
	name, _ := c.Get(middleware.CtxUsername)
	success(c, gin.H{"id": id, "username": name})
}

func (h *Handler) Password(c *gin.Context) {
	id, ok := currentUID(c)
	if !ok {
		return
	}
	var in passwordInput
	if !bindJSON(c, &in) {
		return
	}
	if len(in.New) < 6 {
		fail(c, http.StatusBadRequest, "新密码至少 6 位")
		return
	}
	var user model.User
	if err := h.db.First(&user, id).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(in.Old)) != nil {
		fail(c, http.StatusBadRequest, "原密码不正确")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.New), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, "密码更新失败")
		return
	}
	if err := h.db.Model(&user).Update("password", string(hash)).Error; err != nil {
		writeDBErr(c, err)
		return
	}
	success(c, gin.H{"updated": true})
}
