package repository

import (
	"acustomer-video-server/internal/model"
	"database/sql"
)

// InsertAsset 落库素材元数据
func InsertAsset(a *model.Asset) (int64, error) {
	res, err := DB.Exec(`INSERT INTO assets
		(user_id, shot_no, type, object_key, preview_url, mime, width, height, duration_ms, size_bytes,
		 qc_status, tags_json, usable_start_ms, usable_end_ms, sort_order, authorized)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		a.UserID, a.ShotNo, a.Type, a.ObjectKey, a.PreviewURL, a.Mime, a.Width, a.Height,
		a.DurationMs, a.SizeBytes, a.QcStatus, a.TagsJSON, a.UsableStartMs, a.UsableEndMs,
		a.SortOrder, boolToInt(a.Authorized))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListAssets 列出用户未删除素材
func ListAssets(userID int64) ([]model.Asset, error) {
	rows, err := DB.Query(`SELECT id, shot_no, type, object_key, preview_url, mime, width, height,
		duration_ms, size_bytes, qc_status, COALESCE(qc_json,''), probe_verified, authorized, sort_order, created_at
		FROM assets WHERE user_id=? AND deleted_at IS NULL ORDER BY sort_order ASC, id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Asset
	for rows.Next() {
		var a model.Asset
		if err := rows.Scan(&a.ID, &a.ShotNo, &a.Type, &a.ObjectKey, &a.PreviewURL, &a.Mime,
			&a.Width, &a.Height, &a.DurationMs, &a.SizeBytes, &a.QcStatus, &a.QcJSON, &a.ProbeVerified,
			&a.Authorized, &a.SortOrder, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAsset 单条查询
func GetAsset(userID, id int64) (*model.Asset, error) {
	var a model.Asset
	err := DB.QueryRow(`SELECT id, user_id, shot_no, type, object_key, preview_url, mime, width, height,
		duration_ms, size_bytes, qc_status, tags_json, sort_order, authorized, created_at
		FROM assets WHERE id=? AND user_id=? AND deleted_at IS NULL`, id, userID).Scan(
		&a.ID, &a.UserID, &a.ShotNo, &a.Type, &a.ObjectKey, &a.PreviewURL, &a.Mime,
		&a.Width, &a.Height, &a.DurationMs, &a.SizeBytes, &a.QcStatus, &a.TagsJSON,
		&a.SortOrder, &a.Authorized, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// UpdateAssetSort 批量更新排序（事务）
func UpdateAssetSort(userID int64, ids []int64) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`UPDATE assets SET sort_order=? WHERE id=? AND user_id=?`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, id := range ids {
		if _, err := stmt.Exec(i, id, userID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SoftDeleteAsset 软删除
func SoftDeleteAsset(userID, id int64) error {
	_, err := DB.Exec(`UPDATE assets SET deleted_at=NOW() WHERE id=? AND user_id=?`, id, userID)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
