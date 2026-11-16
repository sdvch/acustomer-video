-- 本项目独立数据库初始化脚本（库名 acustomer_video）。
-- 强制隔离：不引用、不读写 storyboard-system 的任何表。

CREATE TABLE IF NOT EXISTS users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  open_id VARCHAR(128) NOT NULL,
  platform VARCHAR(20) NOT NULL DEFAULT 'wechat',
  nickname VARCHAR(128) DEFAULT '',
  member_tier VARCHAR(20) DEFAULT 'free',
  month_quota INT NOT NULL DEFAULT 30,
  used_quota INT NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_openid_platform (open_id, platform)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS quota_ledger (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  delta INT NOT NULL,
  reason VARCHAR(64) DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS industries (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  name VARCHAR(64) NOT NULL,
  sort_order INT DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS assets (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  shot_no INT NOT NULL DEFAULT 0,
  type TINYINT NOT NULL COMMENT '1=image 2=video',
  object_key VARCHAR(512) NOT NULL,
  preview_url VARCHAR(1024) DEFAULT '',
  mime VARCHAR(64) DEFAULT '',
  width INT DEFAULT 0,
  height INT DEFAULT 0,
  duration_ms BIGINT DEFAULT 0,
  size_bytes BIGINT DEFAULT 0,
  qc_status VARCHAR(20) DEFAULT 'pending',
  tags_json TEXT,
  usable_start_ms BIGINT DEFAULT 0,
  usable_end_ms BIGINT DEFAULT 0,
  sort_order INT DEFAULT 0,
  authorized TINYINT(1) NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at DATETIME DEFAULT NULL,
  KEY idx_user (user_id, sort_order),
  KEY idx_object (object_key(191))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS compose_projects (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  title VARCHAR(255) DEFAULT '',
  industry_id BIGINT DEFAULT 0,
  aspect_ratio VARCHAR(10) DEFAULT '9:16',
  target_width INT DEFAULT 1080,
  target_height INT DEFAULT 1920,
  target_fps INT DEFAULT 30,
  status VARCHAR(20) DEFAULT 'draft',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_user (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS videos (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  project_id BIGINT NOT NULL,
  job_id BIGINT NOT NULL,
  object_key VARCHAR(512) NOT NULL,
  file_url VARCHAR(1024) DEFAULT '',
  poster_url VARCHAR(1024) DEFAULT '',
  duration_ms BIGINT DEFAULT 0,
  width INT DEFAULT 0,
  height INT DEFAULT 0,
  fps INT DEFAULT 0,
  bitrate_bps BIGINT DEFAULT 0,
  codec VARCHAR(32) DEFAULT '',
  subtitle_track_json TEXT,
  bgm_url VARCHAR(1024) DEFAULT '',
  accept_status VARCHAR(20) DEFAULT 'pending',
  publish_status VARCHAR(20) DEFAULT 'none',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_project (project_id),
  KEY idx_job (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS compose_jobs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  project_id BIGINT NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'queued',
  progress INT NOT NULL DEFAULT 0,
  stage VARCHAR(40) DEFAULT '',
  claim_token VARCHAR(64) DEFAULT '',
  heartbeat_at DATETIME DEFAULT NULL,
  input_json TEXT,
  result_video_id BIGINT DEFAULT 0,
  error_message TEXT,
  retry_count INT DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_claim_queue (status, id),
  KEY idx_project (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 以下为 SOP 预留对象表（MVP 不产数据，仅保留结构）
CREATE TABLE IF NOT EXISTS store_info_cards (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  data_json TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS teardown_reports (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  data_json TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS copy_versions (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  project_id BIGINT NOT NULL,
  data_json TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS voice_assets (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL,
  object_key VARCHAR(512) DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO industries (name, sort_order) VALUES
  ('餐饮', 1), ('美业', 2), ('零售', 3), ('健身', 4), ('教育', 5), ('其他', 99)
ON DUPLICATE KEY UPDATE name=name;
