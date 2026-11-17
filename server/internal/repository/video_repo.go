package repository

import (
	"database/sql"

	"acustomer-video-server/internal/model"
)

// InsertVideo 落库成片
func InsertVideo(v *model.Video) (int64, error) {
	res, err := DB.Exec(`INSERT INTO videos
		(project_id, job_id, object_key, file_url, poster_url, duration_ms, width, height, fps, bitrate_bps, codec)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		v.ProjectID, v.JobID, v.ObjectKey, v.FileURL, v.PosterURL, v.DurationMs,
		v.Width, v.Height, v.FPS, v.BitrateBps, v.Codec)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetVideo 查询成片
func GetVideo(userID, id int64) (*model.Video, error) {
	var v model.Video
	err := DB.QueryRow(`SELECT v.id, v.project_id, v.job_id, v.object_key, v.file_url, v.duration_ms,
		v.width, v.height, v.fps, v.bitrate_bps, v.codec, v.created_at
		FROM videos v JOIN compose_projects p ON p.id=v.project_id
		WHERE v.id=? AND p.user_id=?`, id, userID).Scan(
		&v.ID, &v.ProjectID, &v.JobID, &v.ObjectKey, &v.FileURL, &v.DurationMs,
		&v.Width, &v.Height, &v.FPS, &v.BitrateBps, &v.Codec, &v.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// ListVideos 用户成片列表
func ListVideos(userID int64) ([]model.Video, error) {
	rows, err := DB.Query(`SELECT v.id, v.project_id, v.job_id, v.object_key, v.file_url, v.duration_ms,
		v.width, v.height, v.fps, v.bitrate_bps, v.codec, v.created_at
		FROM videos v JOIN compose_projects p ON p.id=v.project_id
		WHERE p.user_id=? ORDER BY v.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.JobID, &v.ObjectKey, &v.FileURL, &v.DurationMs,
			&v.Width, &v.Height, &v.FPS, &v.BitrateBps, &v.Codec, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteVideo 删除成片
func DeleteVideo(userID, id int64) error {
	_, err := DB.Exec(`DELETE v FROM videos v JOIN compose_projects p ON p.id=v.project_id
		WHERE v.id=? AND p.user_id=?`, id, userID)
	return err
}

// AdminDeleteVideo 管理端下架成片（不限用户）
func AdminDeleteVideo(id int64) error {
	_, err := DB.Exec(`DELETE FROM videos WHERE id=?`, id)
	return err
}

// AdminListUserVideos 管理端：指定用户的成片列表（含封面）
func AdminListUserVideos(userID int64) ([]model.Video, error) {
	rows, err := DB.Query(`SELECT v.id, v.project_id, v.job_id, v.object_key, v.file_url, v.duration_ms,
		v.width, v.height, v.fps, v.bitrate_bps, v.codec, v.created_at
		FROM videos v JOIN compose_projects p ON p.id=v.project_id
		WHERE p.user_id=? ORDER BY v.id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Video
	for rows.Next() {
		var v model.Video
		if err := rows.Scan(&v.ID, &v.ProjectID, &v.JobID, &v.ObjectKey, &v.FileURL, &v.DurationMs,
			&v.Width, &v.Height, &v.FPS, &v.BitrateBps, &v.Codec, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
