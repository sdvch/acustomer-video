package repository

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"acustomer-video-server/config"
	_ "github.com/go-sql-driver/mysql"
)

// DB 全局数据库连接
var DB *sql.DB

// Init 连接数据库并自动执行 migrations/ 下的 SQL 迁移（幂等）
func Init(cfg *config.Config) error {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err = db.Ping(); err != nil {
		return fmt.Errorf("连接数据库失败(检查 DB_HOST/DB_PORT/账号): %w", err)
	}
	DB = db

	// 二次隔离校验：确认当前连接的库就是本项目库
	var dbName string
	if err = db.QueryRow("SELECT DATABASE()").Scan(&dbName); err != nil {
		return err
	}
	if !strings.EqualFold(dbName, cfg.DBName) {
		return fmt.Errorf("隔离校验失败: 当前库=%q 期望=%q", dbName, cfg.DBName)
	}

	if err := applyMigrations(db, "migrations"); err != nil {
		return err
	}
	return ensureSchemaUpgrades(db)
}

// ensureSchemaUpgrades 幂等补齐后续版本新增的列（MySQL 无 ADD COLUMN IF NOT EXISTS）。
func ensureSchemaUpgrades(db *sql.DB) error {
	cols := []struct {
		table, column, ddl string
	}{
		{"users", "union_id", "ALTER TABLE users ADD COLUMN union_id VARCHAR(128) DEFAULT ''"},
		{"users", "last_login_at", "ALTER TABLE users ADD COLUMN last_login_at DATETIME DEFAULT NULL"},
		{"assets", "qc_json", "ALTER TABLE assets ADD COLUMN qc_json TEXT"},
		{"assets", "probe_verified", "ALTER TABLE assets ADD COLUMN probe_verified TINYINT(1) NOT NULL DEFAULT 0"},
	}
	for _, c := range cols {
		var n int
		err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema=DATABASE() AND table_name=? AND column_name=?`, c.table, c.column).Scan(&n)
		if err != nil {
			return fmt.Errorf("检查列 %s.%s 失败: %w", c.table, c.column, err)
		}
		if n > 0 {
			continue
		}
		if _, err := db.Exec(c.ddl); err != nil {
			return fmt.Errorf("新增列 %s.%s 失败: %w", c.table, c.column, err)
		}
	}
	return nil
}

func applyMigrations(db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("读取 migrations 目录失败: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		content, err := os.ReadFile(dir + "/" + e.Name())
		if err != nil {
			return err
		}
		statements := splitSQL(string(content))
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err = db.Exec(stmt); err != nil {
				return fmt.Errorf("执行迁移 %s 失败: %w | SQL: %.80s", e.Name(), err, stmt)
			}
		}
	}
	return nil
}

// splitSQL 按分号拆分 SQL（本项目 migration 无存储过程/触发器，简单拆分即可）；
// 过滤纯注释行。
func splitSQL(content string) []string {
	lines := strings.Split(content, "\n")
	var cleaned []string
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "--") {
			continue
		}
		cleaned = append(cleaned, line)
	}
	return strings.Split(strings.Join(cleaned, "\n"), ";")
}

// Close 关闭数据库连接
func Close() {
	if DB != nil {
		_ = DB.Close()
	}
}
