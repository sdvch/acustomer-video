package repository

import (
	"database/sql"
	"fmt"

	"acustomer-video-server/internal/model"
)

// UpsertUser 按 (open_id, platform) 落用户，返回用户 ID 与是否新建。
// 复用已有记录（同一个微信用户多次登录不会重复建号）。
func UpsertUser(openID, unionID, platform, nickname string) (int64, bool, error) {
	if platform == "" {
		platform = "wechat"
	}
	// LAST_INSERT_ID(id) 让重复键时也能拿到已有 id
	res, err := DB.Exec(`INSERT INTO users (open_id, union_id, platform, nickname, last_login_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id), last_login_at=NOW(),
			union_id=IF(VALUES(union_id)<>'', VALUES(union_id), union_id),
			nickname=IF(VALUES(nickname)<>'', VALUES(nickname), nickname)`,
		openID, unionID, platform, nickname)
	if err != nil {
		return 0, false, fmt.Errorf("写入用户失败: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, false, err
	}
	// MySQL: 新插入 affected=1，命中重复键更新 affected=2（值未变时可能为 0）
	affected, _ := res.RowsAffected()
	return id, affected == 1, nil
}

// GetUser 按 ID 查用户
func GetUser(id int64) (*model.User, error) {
	var u model.User
	err := DB.QueryRow(`SELECT id, open_id, platform, nickname, phone, password_hash, status,
		member_tier, month_quota, used_quota, created_at
		FROM users WHERE id=?`, id).Scan(&u.ID, &u.OpenID, &u.Platform, &u.Nickname,
		&u.Phone, &u.PasswordHash, &u.Status,
		&u.MemberTier, &u.MonthQuota, &u.UsedQuota, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// UserDisabled 用户是否被禁用（PK 点查；DB 未初始化时视为未禁用，便于单测）
func UserDisabled(id int64) bool {
	if DB == nil {
		return false
	}
	var status int
	if err := DB.QueryRow(`SELECT status FROM users WHERE id=?`, id).Scan(&status); err != nil {
		return false // 查不到（已注销/异常）不拦截，交由后续业务判断
	}
	return status == model.UserBlocked
}

// GetUserByPhone 按手机号查用户（手机号认证登录用）
func GetUserByPhone(phone string) (*model.User, error) {
	var u model.User
	err := DB.QueryRow(`SELECT id, open_id, platform, nickname, phone, password_hash, status,
		member_tier, month_quota, used_quota, created_at
		FROM users WHERE phone=?`, phone).Scan(&u.ID, &u.OpenID, &u.Platform, &u.Nickname,
		&u.Phone, &u.PasswordHash, &u.Status,
		&u.MemberTier, &u.MonthQuota, &u.UsedQuota, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUserWithPhone 手机号+密码注册
func CreateUserWithPhone(phone, passwordHash, nickname string) (int64, error) {
	res, err := DB.Exec(`INSERT INTO users (open_id, platform, nickname, phone, password_hash, last_login_at)
		VALUES (?, 'phone', ?, ?, ?, NOW())`,
		"phone:"+phone, nickname, phone, passwordHash)
	if err != nil {
		return 0, fmt.Errorf("创建用户失败: %w", err)
	}
	return res.LastInsertId()
}

// BindPhone 微信用户绑定手机号并设置密码
func BindPhone(userID int64, phone, passwordHash string) error {
	_, err := DB.Exec(`UPDATE users SET phone=?, password_hash=? WHERE id=?`, phone, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("绑定手机号失败: %w", err)
	}
	return nil
}

// SetUserStatus 禁用/启用用户
func SetUserStatus(id int64, status int) error {
	_, err := DB.Exec(`UPDATE users SET status=? WHERE id=?`, status, id)
	if err != nil {
		return fmt.Errorf("更新用户状态失败: %w", err)
	}
	return nil
}

// UpdateUserPassword 管理员重置用户密码
func UpdateUserPassword(id int64, passwordHash string) error {
	_, err := DB.Exec(`UPDATE users SET password_hash=? WHERE id=?`, passwordHash, id)
	if err != nil {
		return fmt.Errorf("重置密码失败: %w", err)
	}
	return nil
}

// AdminUserItem 管理后台用户列表行（带项目/成片计数）
type AdminUserItem struct {
	model.User
	ProjectCount int `json:"project_count"`
	VideoCount   int `json:"video_count"`
}

// AdminListUsers 管理后台：关键词（手机号/昵称）搜索 + 分页，附项目与成片计数
func AdminListUsers(keyword string, offset, limit int) ([]AdminUserItem, int, error) {
	where := "1=1"
	args := []interface{}{}
	if keyword != "" {
		where = "(phone LIKE ? OR nickname LIKE ?)"
		kw := "%" + keyword + "%"
		args = append(args, kw, kw)
	}
	var total int
	if err := DB.QueryRow("SELECT COUNT(*) FROM users WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := DB.Query(`SELECT u.id, u.open_id, u.platform, u.nickname, u.phone, u.status,
		u.member_tier, u.month_quota, u.used_quota, u.created_at,
		(SELECT COUNT(*) FROM compose_projects p WHERE p.user_id=u.id) AS project_count,
		(SELECT COUNT(*) FROM videos v WHERE v.project_id IN (SELECT id FROM compose_projects WHERE user_id=u.id)) AS video_count
		FROM users u WHERE `+where+` ORDER BY u.id DESC LIMIT ? OFFSET ?`,
		append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []AdminUserItem
	for rows.Next() {
		var it AdminUserItem
		var phone sql.NullString
		if err := rows.Scan(&it.ID, &it.OpenID, &it.Platform, &it.Nickname, &phone, &it.Status,
			&it.MemberTier, &it.MonthQuota, &it.UsedQuota, &it.CreatedAt,
			&it.ProjectCount, &it.VideoCount); err != nil {
			return nil, 0, err
		}
		if phone.Valid {
			it.Phone = phone.String
		}
		list = append(list, it)
	}
	return list, total, rows.Err()
}
