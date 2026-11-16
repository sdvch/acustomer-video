package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/model"
	"acustomer-video-server/internal/repository"
	"acustomer-video-server/internal/service"
	"github.com/gin-gonic/gin"
)

type ProjectHandler struct {
	cfg *config.Config
	oss *service.OSSService
}

func NewProjectHandler(cfg *config.Config, oss *service.OSSService) *ProjectHandler {
	return &ProjectHandler{cfg: cfg, oss: oss}
}

// CreateProject 创建项目 {title,industry_id} -> {projectId}
func (h *ProjectHandler) Create(c *gin.Context) {
	var req struct {
		Title      string `json:"title"`
		IndustryID int64  `json:"industry_id"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.Title == "" {
		req.Title = "未命名作品"
	}
	id, err := repository.CreateProject(currentUserID(c), req.Title, req.IndustryID)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "创建项目失败")
		return
	}
	ok(c, gin.H{"project_id": id})
}

// Detail 项目详情（含素材与最新成片）
func (h *ProjectHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "id 非法")
		return
	}
	uid := currentUserID(c)
	p, err := repository.GetProject(uid, id)
	if err != nil || p == nil {
		fail(c, http.StatusNotFound, 404, "项目不存在")
		return
	}
	assets, _ := repository.ListAssets(uid)
	videos, _ := repository.ListVideos(uid)
	var latest interface{}
	for _, v := range videos {
		if v.ProjectID == id {
			v.FileURL = h.oss.PublicURL(v.ObjectKey)
			latest = v
			break
		}
	}
	ok(c, gin.H{"project": p, "assets": assets, "latest_video": latest})
}

// List 项目列表
func (h *ProjectHandler) List(c *gin.Context) {
	items, err := repository.ListProjects(currentUserID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询失败")
		return
	}
	if items == nil {
		items = []map[string]interface{}{}
	}
	ok(c, items)
}

// Compose 发起合成 {asset_ids[],duration_policy,clip_seconds,fit_mode} -> {jobId}
// 只写 job 立即返回，实际合成由 worker 异步处理
func (h *ProjectHandler) Compose(c *gin.Context) {
	pid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "id 非法")
		return
	}
	uid := currentUserID(c)
	p, _ := repository.GetProject(uid, pid)
	if p == nil {
		fail(c, http.StatusNotFound, 404, "项目不存在")
		return
	}
	var req struct {
		AssetIDs       []int64 `json:"asset_ids"`
		DurationPolicy string  `json:"duration_policy"`
		ClipSeconds    int     `json:"clip_seconds"`
		FitMode        string  `json:"fit_mode"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.AssetIDs) == 0 {
		fail(c, http.StatusBadRequest, 400, "asset_ids 不能为空")
		return
	}
	if req.ClipSeconds <= 0 {
		req.ClipSeconds = h.cfg.DefaultClipSeconds
	}
	if req.ClipSeconds < h.cfg.MinClipSeconds {
		req.ClipSeconds = h.cfg.MinClipSeconds
	}
	if req.ClipSeconds > h.cfg.MaxClipSeconds {
		req.ClipSeconds = h.cfg.MaxClipSeconds
	}
	if req.FitMode == "" {
		req.FitMode = "pad"
	}

	// ---- 素材校验：归属、存在、授权 ----
	assets, err := repository.GetAssetsByIDs(uid, req.AssetIDs)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询素材失败")
		return
	}
	if len(assets) != len(req.AssetIDs) {
		fail(c, http.StatusBadRequest, 400, "部分素材不存在或已被删除，请返回重新选择")
		return
	}
	for _, a := range assets {
		if !a.Authorized {
			fail(c, http.StatusBadRequest, 400, "存在未勾选授权的素材，无法合成")
			return
		}
	}
	if len(assets) < h.cfg.MinAssetCount || len(assets) > h.cfg.MaxAssetCount {
		fail(c, http.StatusBadRequest, 400,
			fmt.Sprintf("素材段数需在 %d-%d 段之间，当前 %d 段", h.cfg.MinAssetCount, h.cfg.MaxAssetCount, len(assets)))
		return
	}

	// ---- 时长治理：按 SOP 25-35s 校验 ----
	segMs, totalMs := service.PlanSegments(assets, int64(req.ClipSeconds)*1000)
	verdict := service.CheckDuration(totalMs, len(assets),
		h.cfg.MinDurationSec, h.cfg.MaxDurationSec, h.cfg.MinClipSeconds, h.cfg.MaxClipSeconds)
	if !verdict.OK {
		c.JSON(http.StatusBadRequest, Resp{
			Code:    4002,
			Message: verdict.Message,
			Data: gin.H{
				"total_ms": totalMs, "min_ms": verdict.MinMs, "max_ms": verdict.MaxMs,
				"asset_count": len(assets), "suggest_clip_ms": verdict.SuggestMs,
			},
		})
		return
	}

	input := model.JobInput{
		AssetIDs: req.AssetIDs, DurationPolicy: req.DurationPolicy,
		ClipSeconds: req.ClipSeconds, FitMode: req.FitMode,
		SegmentMs: segMs, TotalMs: totalMs,
	}
	inputJSON, _ := json.Marshal(input)

	// 旧任务置 superseded，防止竞态
	_ = repository.SupersedeJobs(pid)
	jobID, err := repository.CreateJob(uid, pid, string(inputJSON))
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "创建合成任务失败")
		return
	}
	ok(c, gin.H{"job_id": jobID})
}

// JobStatus 查询任务进度
func (h *ProjectHandler) JobStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "id 非法")
		return
	}
	job, err := repository.GetJob(currentUserID(c), id)
	if err != nil || job == nil {
		fail(c, http.StatusNotFound, 404, "任务不存在")
		return
	}
	data := gin.H{
		"job_id": job.ID, "status": job.Status, "progress": job.Progress,
		"stage": job.Stage, "error_message": job.ErrorMessage, "result_video_id": job.ResultVideoID,
	}
	if job.Status == model.JobSucceeded && job.ResultVideoID > 0 {
		if v, _ := repository.GetVideo(currentUserID(c), job.ResultVideoID); v != nil {
			v.FileURL = h.oss.PublicURL(v.ObjectKey)
			data["result"] = v
		}
	}
	ok(c, data)
}

// ListVideos 成片列表
func (h *ProjectHandler) ListVideos(c *gin.Context) {
	videos, err := repository.ListVideos(currentUserID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询失败")
		return
	}
	for i := range videos {
		videos[i].FileURL = h.oss.PublicURL(videos[i].ObjectKey)
	}
	if videos == nil {
		videos = []model.Video{}
	}
	ok(c, videos)
}

// DownloadURL 成片签名下载 URL
func (h *ProjectHandler) DownloadURL(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "id 非法")
		return
	}
	v, _ := repository.GetVideo(currentUserID(c), id)
	if v == nil {
		fail(c, http.StatusNotFound, 404, "成片不存在")
		return
	}
	u, err := h.oss.SignGetURL(v.ObjectKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "生成下载链接失败")
		return
	}
	ok(c, gin.H{"url": u, "filename": "video-" + strconv.FormatInt(id, 10) + ".mp4"})
}

// DeleteVideo 删除成片
func (h *ProjectHandler) DeleteVideo(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "id 非法")
		return
	}
	if err := repository.DeleteVideo(currentUserID(c), id); err != nil {
		fail(c, http.StatusInternalServerError, 500, "删除失败")
		return
	}
	ok(c, gin.H{"deleted": id})
}
