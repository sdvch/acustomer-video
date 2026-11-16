# video-output

## Purpose

成片输出规格与出口断言：1080×1920/30fps/H.264 High@4.1/12Mbps 真 CBR/AAC 192k/48kHz/+faststart、时长 25-35s，逐项断言不达标即失败不上传，本规格为数值边界唯一权威源。

## Requirements

### Requirement: 成片目标规格

系统 SHALL 产出 1080×1920（9:16 竖屏）、30fps、H.264 High Profile Level 4.1、12Mbps 真 CBR（b:v=minrate=maxrate=12M、bufsize=24M、nal-hrd=cbr:force-cfr=1）、AAC 192kbps/48kHz 立体声、yuv420p、MP4 +faststart 的成片。

#### Scenario: 成片规格达标

- **WHEN** 一组合规素材完成合成
- **THEN** ffprobe 实测各项参数符合上述规格，码率在 8-15Mbps 区间内视为达标。

### Requirement: 出口断言不达标即失败

系统 SHALL 在任务完成前用 ffmpeg.Validate 逐项校验分辨率、帧率、编码、Profile、Level、码率、音频与时长（25-35s，容差 ±1.5s），任一项不达标 MUST 将任务置为 failed 且不上传成品。

#### Scenario: 断言失败阻断上传

- **WHEN** 合成产物码率越界或时长不在 25-35s
- **THEN** 任务标记 failed，错误信息包含不达标项与实测值，不产生可访问的成品记录。

### Requirement: 片段归一化后无损拼接

系统 SHALL 先将每张图片/每段视频 normalize 为同规格片段（图片用 `-loop 1 -framerate 30` + `-t clipSeconds`，视频统一 scale/pad、fps=30、aresample=48000），再以 concat demuxer + `-c copy` 无损拼接。

#### Scenario: 异构素材统一规格

- **WHEN** 用户混用不同分辨率/帧率的图片与视频素材
- **THEN** 各片段归一化后参数一致，拼接过程无重编码损耗。

### Requirement: 失败原因可观测

ffmpeg 执行失败时系统 SHALL 透出 stderr 尾部（惯例 tail 800 字符），不得只报文件不存在类信息。

#### Scenario: 坏素材失败诊断

- **WHEN** 某素材编解码失败导致任务失败
- **THEN** 任务错误信息包含该素材标识与 ffmpeg stderr 尾部，可据此定位原因。

### Requirement: 字幕与 BGM 的接入点已预留

系统 SHALL 在流水线 normalize→concat→output 的阶段化结构中预留 subtitle（ASS 烧录、思源黑体 Bold、白字黑描边、画面下 1/4 居中、单行≤12 字）与 bgm（amix 混音、音量 ducking）接入位，`video` 表对应字段已预留。

#### Scenario: 本期不含字幕与 BGM

- **WHEN** 当前版本执行合成
- **THEN** subtitle/bgm 阶段跳过，行为与既有成片规格一致。
