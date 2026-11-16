package repository

import (
	"database/sql"
	"fmt"
	"time"

	"acustomer-video-server/internal/model"
)

// CreateJob 插入 queued 任务，返回 jobId
func CreateJob(userID, projectID int64, inputJSON string) (int64, error) {
	res, err := DB.Exec(`INSERT INTO compose_jobs (user_id, project_id, status, input_json, progress, stage)
		VALUES (?,?,?,?,0,'queued')`, userID, projectID, model.JobQueued, inputJSON)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ClaimJob 抢占一个 queued 任务（事务 + FOR UPDATE），返回任务与令牌
func ClaimJob() (*model.ComposeJob, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var job model.ComposeJob
	// SKIP LOCKED：多 worker 并发领取时直接跳过已被锁定的行，避免互相等待产生死锁
	err = tx.QueryRow(`SELECT id, user_id, project_id, input_json, status
		FROM compose_jobs WHERE status=? ORDER BY id ASC LIMIT 1 FOR UPDATE SKIP LOCKED`, model.JobQueued).Scan(
		&job.ID, &job.UserID, &job.ProjectID, &job.InputJSON, &job.Status)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	token := fmt.Sprintf("%d-%d", job.ID, time.Now().UnixNano())
	res, err := tx.Exec(`UPDATE compose_jobs SET status=?, claim_token=?, heartbeat_at=NOW(), progress=0, stage='claimed'
		WHERE id=? AND status=?`, model.JobClaimed, token, job.ID, model.JobQueued)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	job.ClaimToken = token
	job.Status = model.JobClaimed
	return &job, nil
}

// UpdateJobProgress 更新进度（校验令牌，防止失效 worker 覆盖）
func UpdateJobProgress(jobID int64, token, stage string, progress int) error {
	_, err := DB.Exec(`UPDATE compose_jobs SET progress=?, stage=?, heartbeat_at=NOW()
		WHERE id=? AND claim_token=? AND status IN (?,?,?)`,
		progress, stage, jobID, token, model.JobClaimed, model.JobGenerating, model.JobUploading)
	return err
}

// MarkJobGenerating 进入生成中
func MarkJobGenerating(jobID int64, token string) error {
	_, err := DB.Exec(`UPDATE compose_jobs SET status=?, stage='generating', heartbeat_at=NOW()
		WHERE id=? AND claim_token=? AND status=?`, model.JobGenerating, jobID, token, model.JobClaimed)
	return err
}

// MarkJobUploading 进入上传中
func MarkJobUploading(jobID int64, token string) error {
	_, err := DB.Exec(`UPDATE compose_jobs SET status=?, stage='uploading', heartbeat_at=NOW()
		WHERE id=? AND claim_token=? AND status=?`, model.JobUploading, jobID, token, model.JobGenerating)
	return err
}

// CompleteJob 任务成功，写入成片 id
func CompleteJob(jobID int64, token string, videoID int64) error {
	_, err := DB.Exec(`UPDATE compose_jobs SET status=?, stage='done', progress=100, result_video_id=?, heartbeat_at=NOW()
		WHERE id=? AND claim_token=? AND status<>?`, model.JobSucceeded, videoID, jobID, token, model.JobSuperseded)
	return err
}

// FailJob 任务失败（错误截断）
func FailJob(jobID int64, token, errMsg string) error {
	if len(errMsg) > 2000 {
		errMsg = errMsg[:2000]
	}
	_, err := DB.Exec(`UPDATE compose_jobs SET status=?, stage='failed', error_message=?, heartbeat_at=NOW()
		WHERE id=? AND claim_token=? AND status<>?`, model.JobFailed, errMsg, jobID, token, model.JobSuperseded)
	return err
}

// Heartbeat 刷心跳
func Heartbeat(jobID int64, token string) error {
	_, err := DB.Exec(`UPDATE compose_jobs SET heartbeat_at=NOW() WHERE id=? AND claim_token=?`, jobID, token)
	return err
}

// RecoverStaleJobs 看门狗：将超租约的 claimed/generating/uploading 任务重置为 queued
func RecoverStaleJobs(leaseSeconds int) (int64, error) {
	res, err := DB.Exec(`UPDATE compose_jobs SET status=?, claim_token='', stage='', heartbeat_at=NULL
		WHERE status IN (?,?,?) AND (heartbeat_at IS NULL OR heartbeat_at < DATE_SUB(NOW(), INTERVAL ? SECOND))`,
		model.JobQueued, model.JobClaimed, model.JobGenerating, model.JobUploading, leaseSeconds)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GetJob 查询任务（含 user 校验）
func GetJob(userID, id int64) (*model.ComposeJob, error) {
	var j model.ComposeJob
	err := DB.QueryRow(`SELECT id, user_id, project_id, status, progress, stage, result_video_id,
		COALESCE(error_message,''), retry_count, created_at, updated_at
		FROM compose_jobs WHERE id=? AND user_id=?`, id, userID).Scan(
		&j.ID, &j.UserID, &j.ProjectID, &j.Status, &j.Progress, &j.Stage, &j.ResultVideoID,
		&j.ErrorMessage, &j.RetryCount, &j.CreatedAt, &j.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &j, nil
}

// SupersedeJobs 将项目旧的活跃任务置为 superseded（重新发起时调用）
func SupersedeJobs(projectID int64) error {
	_, err := DB.Exec(`UPDATE compose_jobs SET status=? WHERE project_id=? AND status IN (?,?,?,?)`,
		model.JobSuperseded, projectID, model.JobQueued, model.JobClaimed, model.JobGenerating, model.JobUploading)
	return err
}
