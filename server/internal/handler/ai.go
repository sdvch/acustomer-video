package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/repository"
	"acustomer-video-server/internal/service/llm"
	"github.com/gin-gonic/gin"
)

// AIHandler 文本类 AI 能力（SOP 阶段五：文案生成）
type AIHandler struct {
	cfg *config.Config
	llm *llm.Client
}

func NewAIHandler(cfg *config.Config) *AIHandler {
	return &AIHandler{cfg: cfg, llm: llm.New(cfg)}
}

type copyResult struct {
	Hooks []struct {
		Text  string `json:"text"`
		Angle string `json:"angle"`
	} `json:"hooks"`
	Body string   `json:"body"`
	CTA  string   `json:"cta"`
	Tips []string `json:"tips"`
}

const copySystemPrompt = `你是一名服务实体门店的短视频获客编导，擅长抖音/视频号的本地生活类内容。
写作要求：
1. 全部用口语化中文，不用书面语、不用广告法禁用词（最、第一、绝对、保证等）。
2. 钩子必须在前 3 秒抓住人：用痛点、反常识、具体数字或本地场景，不要空喊口号。
3. 字幕单行不超过 12 个字，句子短，便于配音。
4. 成片时长 25-35 秒，口播正文 100-140 字。
5. 不要编造门店没有的信息；缺少信息时用中性表达，不虚构价格和承诺。
只输出 JSON，不要任何解释文字。`

// GenerateCopy 生成获客文案：3 条钩子 + 正文 + CTA
func (h *AIHandler) GenerateCopy(c *gin.Context) {
	var req struct {
		ProjectID     int64    `json:"project_id"`
		Title         string   `json:"title"`
		Industry      string   `json:"industry"`
		StoreName     string   `json:"store_name"`
		SellingPoints []string `json:"selling_points"`
		AssetCount    int      `json:"asset_count"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, 400, "参数错误")
		return
	}
	if !h.llm.Enabled() {
		fail(c, http.StatusServiceUnavailable, 5001, "未配置 DEEPSEEK_API_KEY，无法生成文案")
		return
	}
	if strings.TrimSpace(req.Industry) == "" {
		req.Industry = "实体门店"
	}
	if req.AssetCount <= 0 {
		req.AssetCount = 6
	}

	user := buildCopyPrompt(req.Title, req.Industry, req.StoreName, req.SellingPoints, req.AssetCount)
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(h.cfg.DeepSeekTimeoutSec)*time.Second)
	defer cancel()

	var out copyResult
	if err := h.llm.ChatJSON(ctx, copySystemPrompt, user, &out, 0.7); err != nil {
		fail(c, http.StatusBadGateway, 5002, "文案生成失败: "+err.Error())
		return
	}
	if len(out.Hooks) == 0 && out.Body == "" {
		fail(c, http.StatusBadGateway, 5002, "文案生成结果为空，请重试")
		return
	}

	// 落 copy_versions（SOP 预留表），便于后续做版本对比
	if req.ProjectID > 0 {
		if p, _ := repository.GetProject(currentUserID(c), req.ProjectID); p != nil {
			if payload, err := json.Marshal(out); err == nil {
				_, _ = repository.DB.Exec(`INSERT INTO copy_versions (project_id, data_json) VALUES (?, ?)`,
					req.ProjectID, string(payload))
			}
		}
	}
	ok(c, out)
}

func buildCopyPrompt(title, industry, storeName string, points []string, assetCount int) string {
	var b strings.Builder
	b.WriteString("请为下面这家门店写一条获客短视频文案。\n")
	if storeName != "" {
		b.WriteString("门店名称：" + storeName + "\n")
	}
	b.WriteString("行业：" + industry + "\n")
	if title != "" {
		b.WriteString("视频主题：" + title + "\n")
	}
	if len(points) > 0 {
		b.WriteString("可提供的真实卖点：" + strings.Join(points, "、") + "\n")
	} else {
		b.WriteString("可提供的真实卖点：未提供（不要编造）\n")
	}
	b.WriteString("成片会由 " + strconv.Itoa(assetCount) + " 段素材拼接，每段 3-5 秒。\n\n")
	b.WriteString(`严格按以下 JSON 结构输出：
{
  "hooks": [
    {"text": "钩子文案（≤20字）", "angle": "这条钩子的角度，如痛点/反常识/数字/场景"},
    {"text": "...", "angle": "..."},
    {"text": "...", "angle": "..."}
  ],
  "body": "正文口播，100-140字，句子短，可分行",
  "cta": "结尾行动引导，≤15字",
  "tips": ["拍摄/剪辑建议1", "建议2"]
}`)
	return b.String()
}
