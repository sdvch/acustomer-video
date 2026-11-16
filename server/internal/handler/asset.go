package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/ffmpeg"
	"acustomer-video-server/internal/model"
	"acustomer-video-server/internal/repository"
	"acustomer-video-server/internal/service"
	"github.com/gin-gonic/gin"
)

// AssetHandler 素材相关接口
type AssetHandler struct {
	oss *service.OSSService
	cfg *config.Config
}

func NewAssetHandler(cfg *config.Config, oss *service.OSSService) *AssetHandler {
	return &AssetHandler{oss: oss, cfg: cfg}
}

// Presign 生成预签名上传 URL + objectKey + assetId；
// OSS 模式返回 upload_url 供客户端 PUT 直传；本地模式 upload_url 为空，客户端改用 /assets/upload。
func (h *AssetHandler) Presign(c *gin.Context) {
	var req struct {
		Filename string `json:"filename"`
		Mime     string `json:"mime"`
		Type     int    `json:"type"`
		ShotNo   int    `json:"shot_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if req.Type != model.AssetImage && req.Type != model.AssetVideo {
		fail(c, http.StatusBadRequest, 400, "type 必须为 1(图片) 或 2(视频)")
		return
	}
	if req.Mime == "" {
		req.Mime = "application/octet-stream"
	}
	uid := currentUserID(c)
	objectKey := h.oss.BuildObjectKey(uid, req.Filename)

	// 先落一条 pending 素材记录
	a := &model.Asset{
		UserID: uid, ShotNo: req.ShotNo, Type: req.Type, ObjectKey: objectKey,
		Mime: req.Mime, QcStatus: "pending", SortOrder: req.ShotNo,
	}
	id, err := repository.InsertAsset(a)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "创建素材记录失败: "+err.Error())
		return
	}

	uploadURL, err := h.oss.SignPutURL(objectKey, req.Mime)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "生成上传URL失败: "+err.Error())
		return
	}

	ok(c, gin.H{
		"asset_id":    id,
		"object_key":  objectKey,
		"upload_url":  uploadURL,
		"method":      "PUT",
		"local_mode":  !h.oss.Enabled(),
		"preview_url": h.oss.PublicURL(objectKey),
	})
}

// Upload 本地降级上传（仅 oss 未配置时使用），multipart file 字段名 file
func (h *AssetHandler) Upload(c *gin.Context) {
	objectKey := c.PostForm("object_key")
	assetIDStr := c.PostForm("asset_id")
	if objectKey == "" || assetIDStr == "" {
		fail(c, http.StatusBadRequest, 400, "缺少 object_key/asset_id")
		return
	}
	assetID, err := strconv.ParseInt(assetIDStr, 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "asset_id 非法")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "缺少文件")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "读取文件失败")
		return
	}
	defer f.Close()
	if err := h.oss.PutLocal(objectKey, io.Reader(f)); err != nil {
		fail(c, http.StatusInternalServerError, 500, "保存文件失败: "+err.Error())
		return
	}
	ok(c, gin.H{"asset_id": assetID, "object_key": objectKey, "preview_url": h.oss.PublicURL(objectKey)})
}

// Confirm 上传完成后确认素材：服务端用 ffprobe 真实探测宽高/时长/编码，
// 覆盖前端上报值并做质检，返回 warnings 供前端提示（不阻断上传，除无法解析外）。
func (h *AssetHandler) Confirm(c *gin.Context) {
	uid := currentUserID(c)
	var req struct {
		AssetID    int64  `json:"asset_id"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		DurMs      int64  `json:"duration_ms"`
		Size       int64  `json:"size_bytes"`
		Tags       string `json:"tags_json"`
		Authorized bool   `json:"authorized"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	a, err := repository.GetAsset(uid, req.AssetID)
	if err != nil || a == nil {
		fail(c, http.StatusNotFound, 404, "素材不存在")
		return
	}
	if !req.Authorized {
		fail(c, http.StatusBadRequest, 400, "必须勾选素材授权后才能使用")
		return
	}

	// ---- 服务端质检：把素材落到本地临时文件后 ffprobe ----
	localPath, cleanup, err := h.materialize(a.ObjectKey)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "读取素材失败: "+err.Error())
		return
	}
	defer cleanup()

	preset := ffmpeg.DefaultPreset(h.cfg.FFmpegPath, h.cfg.FFprobePath)
	info, probeErr := preset.Inspect(localPath)
	if probeErr != nil {
		_, _ = repository.DB.Exec(`UPDATE assets SET qc_status='failed', qc_json=?, size_bytes=?, authorized=1 WHERE id=? AND user_id=?`,
			toJSON([]string{probeErr.Error()}), req.Size, req.AssetID, uid)
		fail(c, http.StatusBadRequest, 4003, "素材无法解析，请更换文件: "+probeErr.Error())
		return
	}

	warnings := h.qcWarnings(a.Type, info)
	status := "passed"
	if len(warnings) > 0 {
		status = "warning"
	}
	size := req.Size
	if size <= 0 {
		size = info.SizeBytes
	}
	_, err = repository.DB.Exec(`UPDATE assets SET width=?,height=?,duration_ms=?,size_bytes=?,
		tags_json=?,qc_status=?,qc_json=?,probe_verified=1,authorized=1 WHERE id=? AND user_id=?`,
		info.Width, info.Height, info.DurationMs, size, req.Tags, status, toJSON(warnings), req.AssetID, uid)
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "确认失败")
		return
	}
	ok(c, gin.H{
		"asset_id":  req.AssetID,
		"qc_status": status,
		"warnings":  warnings,
		"probe": gin.H{
			"kind": info.Kind, "width": info.Width, "height": info.Height,
			"duration_ms": info.DurationMs, "codec": info.Codec,
			"fps": info.FPS, "has_audio": info.HasAudio, "size_bytes": size,
		},
	})
}

// materialize 把素材准备成本地可读路径（OSS 模式下载到临时目录），返回清理函数
func (h *AssetHandler) materialize(objectKey string) (string, func(), error) {
	if !h.oss.Enabled() {
		p := h.oss.LocalPath(objectKey)
		if _, err := os.Stat(p); err != nil {
			return "", func() {}, fmt.Errorf("文件不存在: %w", err)
		}
		return p, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "acustomer-qc-")
	if err != nil {
		return "", func() {}, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	dest := filepath.Join(dir, "asset"+path.Ext(objectKey))
	if err := h.oss.DownloadToLocal(objectKey, dest); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return dest, cleanup, nil
}

// qcWarnings 按 SOP 拍摄规范给出素材问题提示（只提示，不拦截）
func (h *AssetHandler) qcWarnings(assetType int, info *ffmpeg.MediaInfo) []string {
	var w []string
	if info.Width <= 0 || info.Height <= 0 {
		return []string{"无法读取素材分辨率"}
	}
	shortSide := info.Width
	if info.Height < shortSide {
		shortSide = info.Height
	}
	if shortSide < 720 {
		w = append(w, fmt.Sprintf("分辨率偏低（%dx%d），建议至少 720p，否则成片会发虚", info.Width, info.Height))
	}
	if info.Height <= info.Width {
		w = append(w, fmt.Sprintf("素材为横屏/方图（%dx%d），将自动补黑边；建议竖屏 9:16 拍摄", info.Width, info.Height))
	} else {
		// 9:16 理想比例 0.5625，偏离 15% 视为画幅不标准
		ratio := float64(info.Width) / float64(info.Height)
		if ratio > 0.5625*1.15 || ratio < 0.5625*0.85 {
			w = append(w, fmt.Sprintf("画幅非标准 9:16（%dx%d），成片会补边或裁切", info.Width, info.Height))
		}
	}
	if assetType == model.AssetVideo {
		switch {
		case info.DurationMs > 0 && info.DurationMs < 2000:
			w = append(w, fmt.Sprintf("镜头仅 %.1f 秒，短于建议的 3-5 秒", float64(info.DurationMs)/1000))
		case info.DurationMs > int64(h.cfg.MaxClipSeconds)*1000:
			w = append(w, fmt.Sprintf("镜头 %.1f 秒偏长，合成时只截取前 %d 秒", float64(info.DurationMs)/1000, h.cfg.MaxClipSeconds))
		}
		if info.Codec != "h264" && info.Codec != "hevc" {
			w = append(w, fmt.Sprintf("编码格式 %s 将统一转码为 H.264", info.Codec))
		}
	}
	return w
}

func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// List 素材列表
func (h *AssetHandler) List(c *gin.Context) {
	assets, err := repository.ListAssets(currentUserID(c))
	if err != nil {
		fail(c, http.StatusInternalServerError, 500, "查询失败")
		return
	}
	// 本地模式下把 object_key 换成可访问的预览 URL
	for i := range assets {
		if assets[i].PreviewURL == "" {
			assets[i].PreviewURL = h.oss.PublicURL(assets[i].ObjectKey)
		}
	}
	ok(c, assets)
}

// Sort 批量排序：{ids:[...]}
func (h *AssetHandler) Sort(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		fail(c, http.StatusBadRequest, 400, "ids 不能为空")
		return
	}
	if err := repository.UpdateAssetSort(currentUserID(c), req.IDs); err != nil {
		fail(c, http.StatusInternalServerError, 500, "排序失败")
		return
	}
	ok(c, gin.H{"updated": len(req.IDs)})
}

// Delete 软删除素材
func (h *AssetHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, 400, "id 非法")
		return
	}
	if err := repository.SoftDeleteAsset(currentUserID(c), id); err != nil {
		fail(c, http.StatusInternalServerError, 500, "删除失败")
		return
	}
	ok(c, gin.H{"deleted": id})
}
