package handler

import (
	"math/rand"
	"net/http"
	"strconv"

	"golang.org/x/crypto/bcrypt"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/middleware"
	"acustomer-video-server/internal/model"
	"acustomer-video-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理后台全部接口（/api/v1/admin/*，经 nginx 反代为 /acv/v1/admin/*）
type AdminHandler struct {
	cfg *config.Config
}

func NewAdminHandler(cfg *config.Config) *AdminHandler {
	return &AdminHandler{cfg: cfg}
}

// ---------- 登录 / 会话 ----------

// Login 管理员登录：成功签发 scope=admin JWT 并写入 httpOnly Cookie
func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || req.Password == "" {
		fail(c, http.StatusBadRequest, 400, "请输入用户名和密码")
		return
	}
	a, _ := repository.GetAdminByUsername(req.Username)
	if a == nil || bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(req.Password)) != nil {
		fail(c, http.StatusUnauthorized, 401, "用户名或密码错误")
		return
	}
	if a.Status != 1 {
		fail(c, http.StatusForbidden, 403, "该管理员已停用")
		return
	}
	token, err := middleware.IssueAdminToken(h.cfg.JWTSecret, a.ID, a.Role, 12)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "签发令牌失败")
		return
	}
	repository.TouchAdminLogin(a.ID)
	// httpOnly Cookie（生产为 http 内网直连 + IP 访问，暂不启用 Secure；上 HTTPS 后开启）
	c.SetCookie("admin_token", token, 12*3600, "/", "", false, true)
	ok(c, gin.H{"id": a.ID, "username": a.Username, "role": a.Role})
}

// Logout 清除管理端 Cookie
func (h *AdminHandler) Logout(c *gin.Context) {
	c.SetCookie("admin_token", "", -1, "/", "", false, true)
	ok(c, gin.H{})
}

// Profile 当前管理员信息
func (h *AdminHandler) Profile(c *gin.Context) {
	a := h.currentAdmin(c)
	ok(c, gin.H{"id": a.ID, "username": a.Username, "role": a.Role})
}

func (h *AdminHandler) currentAdmin(c *gin.Context) *model.Admin {
	id := c.GetInt64("adminID")
	a, _ := repository.GetAdmin(id)
	if a == nil {
		return &model.Admin{ID: id, Role: c.GetString("adminRole")}
	}
	return a
}

// requireSuper 超管专属接口的权限校验
func (h *AdminHandler) requireSuper(c *gin.Context) bool {
	if c.GetString("adminRole") == model.AdminRoleSuper {
		return true
	}
	fail(c, http.StatusForbidden, 403, "需要超级管理员权限")
	return false
}

// ---------- 用户管理 ----------

// ListUsers 用户列表：关键词（手机号/昵称）+ 分页，附项目/成片计数
func (h *AdminHandler) ListUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	list, total, err := repository.AdminListUsers(keyword, (page-1)*size, size)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询失败: "+err.Error())
		return
	}
	if list == nil {
		list = []repository.AdminUserItem{}
	}
	ok(c, gin.H{"list": list, "total": total, "page": page, "size": size})
}

// UserDetail 用户详情
func (h *AdminHandler) UserDetail(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	u, err := repository.GetUser(id)
	if err != nil || u == nil {
		fail(c, http.StatusNotFound, 404, "用户不存在")
		return
	}
	u.OpenID = maskOpenID(u.OpenID)
	u.PasswordHash = ""
	ok(c, u)
}

// SetUserStatus 禁用/启用用户
func (h *AdminHandler) SetUserStatus(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		Status int `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || (req.Status != 0 && req.Status != 1) {
		fail(c, http.StatusBadRequest, 400, "参数错误（status 取 0 或 1）")
		return
	}
	if u, _ := repository.GetUser(id); u == nil {
		fail(c, http.StatusNotFound, 404, "用户不存在")
		return
	}
	if err := repository.SetUserStatus(id, req.Status); err != nil {
		fail(c, http.StatusInternalServerError, 500, "操作失败: "+err.Error())
		return
	}
	ok(c, gin.H{"id": id, "status": req.Status})
}

// ResetUserPassword 重置用户密码：服务端生成随机密码返回一次
func (h *AdminHandler) ResetUserPassword(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if u, _ := repository.GetUser(id); u == nil {
		fail(c, http.StatusNotFound, 404, "用户不存在")
		return
	}
	newPwd := randomAdminPassword(10)
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "重置失败")
		return
	}
	if err := repository.UpdateUserPassword(id, string(hash)); err != nil {
		fail(c, http.StatusInternalServerError, 500, "重置失败: "+err.Error())
		return
	}
	ok(c, gin.H{"new_password": newPwd})
}

// UserProjects 用户的项目与成片（内容审查）
func (h *AdminHandler) UserProjects(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if u, _ := repository.GetUser(id); u == nil {
		fail(c, http.StatusNotFound, 404, "用户不存在")
		return
	}
	projects, err := repository.ListProjects(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询失败: "+err.Error())
		return
	}
	videos, err := repository.AdminListUserVideos(id)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询失败: "+err.Error())
		return
	}
	if projects == nil {
		projects = []map[string]interface{}{}
	}
	if videos == nil {
		videos = []model.Video{}
	}
	ok(c, gin.H{"projects": projects, "videos": videos})
}

// DeleteVideo 下架成片
func (h *AdminHandler) DeleteVideo(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if err := repository.AdminDeleteVideo(id); err != nil {
		fail(c, http.StatusInternalServerError, 500, "下架失败: "+err.Error())
		return
	}
	ok(c, gin.H{"id": id})
}

// ---------- 数据总览 ----------

func (h *AdminHandler) StatsOverview(c *gin.Context) {
	s, err := repository.GetStatsOverview()
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "统计失败: "+err.Error())
		return
	}
	ok(c, s)
}

// ---------- 管理员账号管理（仅超管） ----------

func (h *AdminHandler) ListAdmins(c *gin.Context) {
	if !h.requireSuper(c) {
		return
	}
	list, err := repository.ListAdmins()
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询失败: "+err.Error())
		return
	}
	if list == nil {
		list = []model.Admin{}
	}
	for i := range list {
		list[i].PasswordHash = ""
	}
	ok(c, list)
}

func (h *AdminHandler) CreateAdmin(c *gin.Context) {
	if !h.requireSuper(c) {
		return
	}
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Username == "" || len(req.Password) < 8 {
		fail(c, http.StatusBadRequest, 400, "用户名必填，密码至少 8 位")
		return
	}
	if req.Role != model.AdminRoleSuper && req.Role != model.AdminRoleOperator {
		req.Role = model.AdminRoleOperator
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "创建失败")
		return
	}
	id, err := repository.CreateAdmin(req.Username, string(hash), req.Role)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "创建失败: "+err.Error())
		return
	}
	ok(c, gin.H{"id": id, "username": req.Username, "role": req.Role})
}

func (h *AdminHandler) UpdateAdmin(c *gin.Context) {
	if !h.requireSuper(c) {
		return
	}
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if a, _ := repository.GetAdmin(id); a == nil {
		fail(c, http.StatusNotFound, 404, "管理员不存在")
		return
	}
	var req struct {
		Role     *string `json:"role"`
		Status   *int    `json:"status"`
		Password *string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if req.Role != nil && *req.Role != model.AdminRoleSuper && *req.Role != model.AdminRoleOperator {
		fail(c, http.StatusBadRequest, 400, "role 取 super 或 operator")
		return
	}
	if req.Password != nil && len(*req.Password) < 8 {
		fail(c, http.StatusBadRequest, 400, "密码至少 8 位")
		return
	}
	if req.Status != nil && (*req.Status != 0 && *req.Status != 1) {
		fail(c, http.StatusBadRequest, 400, "status 取 0 或 1")
		return
	}
	var hash *string
	if req.Password != nil {
		hb, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "更新失败")
			return
		}
		s := string(hb)
		hash = &s
	}
	if err := repository.UpdateAdmin(id, req.Role, req.Status, hash); err != nil {
		fail(c, http.StatusInternalServerError, 500, "更新失败: "+err.Error())
		return
	}
	ok(c, gin.H{"id": id})
}

// randomAdminPassword 管理端重置密码用（去掉易混字符）
func randomAdminPassword(n int) string {
	const chars = "23456789abcdefghjkmnpqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
