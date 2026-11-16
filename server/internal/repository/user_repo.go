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
	err := DB.QueryRow(`SELECT id, open_id, platform, nickname, member_tier, month_quota, used_quota, created_at
		FROM users WHERE id=?`, id).Scan(&u.ID, &u.OpenID, &u.Platform, &u.Nickname,
		&u.MemberTier, &u.MonthQuota, &u.UsedQuota, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
