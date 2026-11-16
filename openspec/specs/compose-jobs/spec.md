# compose-jobs

## Purpose

合成任务的时长治理、并发认领与进度可观测：提交前强制 25-35s 时长校验，多 worker 用 FOR UPDATE SKIP LOCKED 无死锁认领，阶段加权进度与失败原因持久化可查。

## Requirements

### Requirement: 合成前时长治理

系统 SHALL 在提交合成（POST /projects/:id/compose）前逐段计算时长求和，总时长不在 25-35s 区间时 MUST 拒绝并给出"每段调到几秒/补几段"的建议。

#### Scenario: 时长越界拒绝

- **WHEN** 项目素材总时长为 18s 或 40s
- **THEN** 拒绝创建合成任务，响应包含具体调整建议。

### Requirement: 多 worker 无死锁认领

系统 SHALL 使用 `FOR UPDATE SKIP LOCKED`（ClaimJob）认领任务，多 worker 并发时不互相阻塞、不重复认领。

#### Scenario: 并发认领互斥

- **WHEN** 两个 worker 同时认领待处理任务
- **THEN** 每个任务仅被一个 worker 认领，无死锁。

### Requirement: 阶段加权进度

系统 SHALL 通过 `-progress pipe:1 -nostats` 解析 out_time_us 上报进度，阶段加权为 transcode 0-70%、concat 70-85%、validate/上传 85-100%，每 2% 写库一次。

#### Scenario: 前端进度可见

- **WHEN** 合成任务执行中
- **THEN** 前端可查询到单调递增的进度百分比，完成时到达 100%。

### Requirement: 任务状态与错误持久化

系统 SHALL 持久化任务状态（pending/processing/succeeded/failed）与失败原因，提供可恢复的任务状态查询。

#### Scenario: 服务重启后状态可查

- **WHEN** worker 处理中服务重启
- **THEN** 任务状态可查询，不会丢失或误报成功。
