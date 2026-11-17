package repository

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"

	"golang.org/x/crypto/bcrypt"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/model"
)

// SeedAdmin admins 表为空时创建首个超管；密码未配置则生成随机密码打日志。
func SeedAdmin(db *sql.DB, cfg *config.Config) error {
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM admins`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	username := cfg.AdminSeedUsername
	if username == "" {
		username = "admin"
	}
	password := cfg.AdminSeedPassword
	if password == "" {
		password = randomPassword(12)
		log.Printf("[种子] 已生成初始管理员 -> 用户名: %s 初始密码: %s （请登录后尽快修改）", username, password)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`INSERT INTO admins (username, password_hash, role) VALUES (?, ?, 'super')`,
		username, string(hash)); err != nil {
		return fmt.Errorf("创建初始管理员失败: %w", err)
	}
	if cfg.AdminSeedPassword != "" {
		log.Printf("[种子] 已创建初始管理员: %s（来自 ADMIN_PASSWORD 环境变量）", username)
	}
	return nil
}

const pwChars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"

func randomPassword(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = pwChars[rand.Intn(len(pwChars))]
	}
	return string(b)
}

// GetAdminByUsername 管理员登录用
func GetAdminByUsername(username string) (*model.Admin, error) {
	var a model.Admin
	err := DB.QueryRow(`SELECT id, username, password_hash, role, status,
		IFNULL(DATE_FORMAT(last_login_at, '%Y-%m-%d %H:%i:%s'), ''), created_at
		FROM admins WHERE username=?`, username).Scan(&a.ID, &a.Username, &a.PasswordHash,
		&a.Role, &a.Status, &a.LastLoginAt, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// GetAdmin 按 ID 查管理员
func GetAdmin(id int64) (*model.Admin, error) {
	var a model.Admin
	err := DB.QueryRow(`SELECT id, username, password_hash, role, status,
		IFNULL(DATE_FORMAT(last_login_at, '%Y-%m-%d %H:%i:%s'), ''), created_at
		FROM admins WHERE id=?`, id).Scan(&a.ID, &a.Username, &a.PasswordHash,
		&a.Role, &a.Status, &a.LastLoginAt, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// TouchAdminLogin 更新最后登录时间
func TouchAdminLogin(id int64) {
	_, _ = DB.Exec(`UPDATE admins SET last_login_at=NOW() WHERE id=?`, id)
}

// ListAdmins 管理员列表
func ListAdmins() ([]model.Admin, error) {
	rows, err := DB.Query(`SELECT id, username, password_hash, role, status,
		IFNULL(DATE_FORMAT(last_login_at, '%Y-%m-%d %H:%i:%s'), ''), created_at
		FROM admins ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []model.Admin
	for rows.Next() {
		var a model.Admin
		if err := rows.Scan(&a.ID, &a.Username, &a.PasswordHash, &a.Role, &a.Status,
			&a.LastLoginAt, &a.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, rows.Err()
}

// CreateAdmin 新增管理员
func CreateAdmin(username, passwordHash, role string) (int64, error) {
	res, err := DB.Exec(`INSERT INTO admins (username, password_hash, role) VALUES (?, ?, ?)`,
		username, passwordHash, role)
	if err != nil {
		return 0, fmt.Errorf("创建管理员失败: %w", err)
	}
	return res.LastInsertId()
}

// UpdateAdmin 更新管理员（角色/状态/密码按需）
func UpdateAdmin(id int64, role *string, status *int, passwordHash *string) error {
	sets, args := "", []interface{}{}
	if role != nil {
		sets += "role=?,"
		args = append(args, *role)
	}
	if status != nil {
		sets += "status=?,"
		args = append(args, *status)
	}
	if passwordHash != nil {
		sets += "password_hash=?,"
		args = append(args, *passwordHash)
	}
	if sets == "" {
		return nil
	}
	sets = sets[:len(sets)-1]
	args = append(args, id)
	_, err := DB.Exec(`UPDATE admins SET `+sets+` WHERE id=?`, args...)
	return err
}

// AdminStatsOverview 运营数据总览
type AdminStatsOverview struct {
	TotalUsers    int   `json:"total_users"`
	TodayUsers    int   `json:"today_users"`
	TotalProjects int   `json:"total_projects"`
	TotalVideos   int   `json:"total_videos"`
	TotalJobs     int   `json:"total_jobs"`
	StorageBytes  int64 `json:"storage_bytes"`
}

// GetStatsOverview 汇总运营指标
func GetStatsOverview() (*AdminStatsOverview, error) {
	s := &AdminStatsOverview{}
	queries := []struct {
		sql  string
		dst  interface{}
		args []interface{}
	}{
		{"SELECT COUNT(*) FROM users WHERE status=1", &s.TotalUsers, nil},
		{"SELECT COUNT(*) FROM users WHERE created_at >= CURDATE()", &s.TodayUsers, nil},
		{"SELECT COUNT(*) FROM compose_projects", &s.TotalProjects, nil},
		{"SELECT COUNT(*) FROM videos", &s.TotalVideos, nil},
		{"SELECT COUNT(*) FROM compose_jobs", &s.TotalJobs, nil},
		{"SELECT IFNULL(SUM(size_bytes),0) FROM assets WHERE deleted_at IS NULL", &s.StorageBytes, nil},
	}
	for _, q := range queries {
		if err := DB.QueryRow(q.sql, q.args...).Scan(q.dst); err != nil {
			return nil, err
		}
	}
	return s, nil
}
