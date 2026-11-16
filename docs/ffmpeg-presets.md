# ffmpeg 合成规格与流水线

## 目标规格（SOP 表10）
| 项 | 值 |
|---|---|
| 分辨率 | 1080×1920（9:16 竖屏） |
| 帧率 | 30fps |
| 编码 | H.264 High Profile level 4.1 |
| 码率 | 12Mbps 真 CBR（b:v=minrate=maxrate=12M, bufsize=24M, `-x264-params nal-hrd=cbr:force-cfr=1`），实测 11.9Mbps |
| 音频 | AAC 192kbps / 48kHz / 立体声 |
| 像素格式 | yuv420p |
| 封装 | MP4 +faststart |
| 时长 | 25-35s（compose 强制校验，出口断言 ±1.5s 容差） |

## 流水线（internal/ffmpeg/pipeline.go）
1. **normalize**：每张图/每段视频转成同规格 MP4 片段
   - 图片：`-loop 1 -framerate 30 -i img -f lavfi -i anullsrc` + scale/pad + `-t clipSeconds`
   - 视频：scale/pad + `fps=30` + `aresample=48000`
   - 适配：`fit_mode=pad`（黑边填充，默认）或 `crop`（裁切填满）
2. **concat**：concat demuxer + `-c copy` 无损拼接（各段参数一致）
3. **validate**：ffprobe 校验分辨率/帧率/编码/Profile/Level/**码率 8-15Mbps**/音频/时长 25-35s，不达标则 job failed 不上传

## 进度
`-progress pipe:1 -nostats` 解析 `out_time_us`，阶段加权：transcode 0-70%、concat 70-85%、validate/上传 85-100%，每 2% 写库一次。

## 下一轮接入点
`pipeline.go` 的 normalize→concat→**subtitle**→**bgm**→output 阶段化结构中，subtitle/bgm 本期跳过。
- 字幕：ASS 烧录，思源黑体 Bold，白字黑描边，画面下 1/4 居中，单行≤12 字
- BGM：`amix` 混音，音量 ducking
- 对应 `video` 表字段 `subtitle_track_json / bgm_url` 已预留
