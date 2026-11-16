package repository

import (
	"database/sql"

	"acustomer-video-server/internal/model"
)

// CreateProject 创建合成项目
func CreateProject(userID int64, title string, industryID int64) (int64, error) {
	res, err := DB.Exec(`INSERT INTO compose_projects (user_id, title, industry_id) VALUES (?,?,?)`, userID, title, industryID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetProject 查询项目
func GetProject(userID, id int64) (*model.ComposeProject, error) {
	var p model.ComposeProject
	err := DB.QueryRow(`SELECT id, user_id, title, industry_id, aspect_ratio, target_width, target_height, target_fps, status, created_at
		FROM compose_projects WHERE id=? AND user_id=?`, id, userID).Scan(
		&p.ID, &p.UserID, &p.Title, &p.IndustryID, &p.AspectRatio, &p.TargetWidth,
		&p.TargetHeight, &p.TargetFPS, &p.Status, &p.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ListProjects 用户项目列表（含最新成片）
func ListProjects(userID int64) ([]map[string]interface{}, error) {
	rows, err := DB.Query(`SELECT p.id, p.title, p.status, p.created_at,
		(SELECT id FROM videos v WHERE v.project_id=p.id ORDER BY v.id DESC LIMIT 1) AS latest_video_id
		FROM compose_projects p WHERE p.user_id=? ORDER BY p.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, latest int64
		var title, status, created string
		var latestNull sql.NullInt64
		if err := rows.Scan(&id, &title, &status, &created, &latestNull); err != nil {
			return nil, err
		}
		if latestNull.Valid {
			latest = latestNull.Int64
		}
		out = append(out, map[string]interface{}{
			"id": id, "title": title, "status": status, "created_at": created, "latest_video_id": latest,
		})
	}
	return out, rows.Err()
}

// GetAssetsByIDs 按 id 列表查询素材（保持传入顺序），用于合成输入快照
func GetAssetsByIDs(userID int64, ids []int64) ([]model.Asset, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := DB.Query(`SELECT id, shot_no, type, object_key, duration_ms, sort_order, authorized
		FROM assets WHERE user_id=? AND deleted_at IS NULL`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byID := map[int64]model.Asset{}
	for rows.Next() {
		var a model.Asset
		if err := rows.Scan(&a.ID, &a.ShotNo, &a.Type, &a.ObjectKey, &a.DurationMs, &a.SortOrder, &a.Authorized); err != nil {
			return nil, err
		}
		byID[a.ID] = a
	}
	var out []model.Asset
	for _, id := range ids {
		if a, ok := byID[id]; ok {
			out = append(out, a)
		}
	}
	return out, nil
}
