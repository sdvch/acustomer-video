# asset-quality

## Purpose

素材上传后的服务端质检：以 ffprobe 实测真实宽高、时长与编码为准，前端上报值仅作参考并会被覆盖，质检警告（横屏、低分辨率、镜头过长）必须明示给用户。

## Requirements

### Requirement: 服务端复核素材真实属性

系统 SHALL 在素材确认时（POST /assets/confirm）用 ffprobe 复核真实宽高、时长与编码，并覆盖前端上报值。

#### Scenario: 前端上报失真

- **WHEN** 前端上报的分辨率或时长与服务端实测不一致
- **THEN** 以服务端实测值为准入判据，前端上报值被覆盖。

### Requirement: 质检警告明示

系统 SHALL 对横屏素材、低分辨率素材、镜头过长素材返回 warnings，质检结果为 passed 的素材方可进入合成。

#### Scenario: 横屏素材警告

- **WHEN** 用户提交宽大于高的视频素材
- **THEN** 素材质检通过但在 warnings 中标注横屏提示，不静默吞掉。
