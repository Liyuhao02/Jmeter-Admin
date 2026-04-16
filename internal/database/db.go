package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"jmeter-admin/config"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB() error {
	dbPath := filepath.Join(config.GlobalConfig.Dirs.Data, "jmeter-admin.db")

	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}

	if err := DB.Ping(); err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 创建表
	if err := createTables(); err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	return nil
}

func createTables() error {
	// scripts 表
	scriptsTable := `
	CREATE TABLE IF NOT EXISTS scripts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		file_path TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	);`
	if _, err := DB.Exec(scriptsTable); err != nil {
		return fmt.Errorf("创建 scripts 表失败: %w", err)
	}

	// script_files 表
	scriptFilesTable := `
	CREATE TABLE IF NOT EXISTS script_files (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id INTEGER NOT NULL,
		file_name TEXT NOT NULL,
		file_path TEXT NOT NULL,
		file_type TEXT NOT NULL,
		created_at DATETIME,
		FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE
	);`
	if _, err := DB.Exec(scriptFilesTable); err != nil {
		return fmt.Errorf("创建 script_files 表失败: %w", err)
	}

	// slaves 表
	slavesTable := `
	CREATE TABLE IF NOT EXISTS slaves (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		host TEXT NOT NULL,
		port INTEGER NOT NULL,
		status TEXT DEFAULT 'offline',
		created_at DATETIME
	);`
	if _, err := DB.Exec(slavesTable); err != nil {
		return fmt.Errorf("创建 slaves 表失败: %w", err)
	}

	// executions 表
	executionsTable := `
	CREATE TABLE IF NOT EXISTS executions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id INTEGER NOT NULL,
		script_name TEXT NOT NULL,
		slave_ids TEXT,
		status TEXT DEFAULT 'running',
		start_time DATETIME,
		end_time DATETIME,
		duration INTEGER DEFAULT 0,
		remarks TEXT,
		save_http_details INTEGER DEFAULT 0,
		include_master INTEGER DEFAULT 0,
		split_csv INTEGER DEFAULT 0,
		result_path TEXT,
		report_path TEXT,
		summary_data TEXT,
		log_path TEXT,
		created_at DATETIME,
		FOREIGN KEY (script_id) REFERENCES scripts(id)
	);`
	if _, err := DB.Exec(executionsTable); err != nil {
		return fmt.Errorf("创建 executions 表失败: %w", err)
	}

	// 数据库迁移：添加新列（如果不存在）
	if err := migrateExecutionsTable(); err != nil {
		return fmt.Errorf("迁移 executions 表失败: %w", err)
	}

	// 迁移 script_files 表：添加 updated_at 列
	if err := migrateScriptFilesTable(); err != nil {
		return fmt.Errorf("迁移 script_files 表失败: %w", err)
	}

	// 迁移 slaves 表：添加 last_check_time 列
	if err := migrateSlavesTable(); err != nil {
		return fmt.Errorf("迁移 slaves 表失败: %w", err)
	}

	// 创建 script_versions 表
	if err := createScriptVersionsTable(); err != nil {
		return fmt.Errorf("创建 script_versions 表失败: %w", err)
	}

	// 创建索引
	if err := createIndexes(); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	return nil
}

// migrateExecutionsTable 执行 executions 表的迁移
func migrateExecutionsTable() error {
	// 检查并添加 duration 列
	var durationExists int
	err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name='duration'").Scan(&durationExists)
	if err == nil && durationExists == 0 {
		if _, err := DB.Exec("ALTER TABLE executions ADD COLUMN duration INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("添加 duration 列失败: %w", err)
		}
	}

	// 检查并添加 remarks 列
	var remarksExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name='remarks'").Scan(&remarksExists)
	if err == nil && remarksExists == 0 {
		if _, err := DB.Exec("ALTER TABLE executions ADD COLUMN remarks TEXT"); err != nil {
			return fmt.Errorf("添加 remarks 列失败: %w", err)
		}
	}

	// 检查并添加 save_http_details 列
	var saveHTTPDetailsExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name='save_http_details'").Scan(&saveHTTPDetailsExists)
	if err == nil && saveHTTPDetailsExists == 0 {
		if _, err := DB.Exec("ALTER TABLE executions ADD COLUMN save_http_details INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("添加 save_http_details 列失败: %w", err)
		}
	}

	// 检查并添加 include_master 列
	var includeMasterExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name='include_master'").Scan(&includeMasterExists)
	if err == nil && includeMasterExists == 0 {
		if _, err := DB.Exec("ALTER TABLE executions ADD COLUMN include_master INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("添加 include_master 列失败: %w", err)
		}
	}

	// 检查并添加 split_csv 列
	var splitCSVExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name='split_csv'").Scan(&splitCSVExists)
	if err == nil && splitCSVExists == 0 {
		if _, err := DB.Exec("ALTER TABLE executions ADD COLUMN split_csv INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("添加 split_csv 列失败: %w", err)
		}
	}

	// 检查并添加 is_baseline 列
	var isBaselineExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('executions') WHERE name='is_baseline'").Scan(&isBaselineExists)
	if err == nil && isBaselineExists == 0 {
		if _, err := DB.Exec("ALTER TABLE executions ADD COLUMN is_baseline INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("添加 is_baseline 列失败: %w", err)
		}
	}

	return nil
}

// migrateScriptFilesTable 迁移 script_files 表
func migrateScriptFilesTable() error {
	var updatedAtExists int
	err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('script_files') WHERE name='updated_at'").Scan(&updatedAtExists)
	if err == nil && updatedAtExists == 0 {
		if _, err := DB.Exec("ALTER TABLE script_files ADD COLUMN updated_at DATETIME"); err != nil {
			return fmt.Errorf("添加 updated_at 列失败: %w", err)
		}
	}
	return nil
}

// migrateSlavesTable 迁移 slaves 表
func migrateSlavesTable() error {
	// 添加 last_check_time 列
	var lastCheckTimeExists int
	err := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='last_check_time'").Scan(&lastCheckTimeExists)
	if err == nil && lastCheckTimeExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN last_check_time DATETIME"); err != nil {
			return fmt.Errorf("添加 last_check_time 列失败: %w", err)
		}
	}

	// 添加 agent_port 列
	var agentPortExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='agent_port'").Scan(&agentPortExists)
	if err == nil && agentPortExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN agent_port INTEGER DEFAULT 8089"); err != nil {
			return fmt.Errorf("添加 agent_port 列失败: %w", err)
		}
	}

	// 添加 agent_token 列
	var agentTokenExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='agent_token'").Scan(&agentTokenExists)
	if err == nil && agentTokenExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN agent_token TEXT DEFAULT ''"); err != nil {
			return fmt.Errorf("添加 agent_token 列失败: %w", err)
		}
	}

	// 添加 agent_status 列
	var agentStatusExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='agent_status'").Scan(&agentStatusExists)
	if err == nil && agentStatusExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN agent_status TEXT DEFAULT 'offline'"); err != nil {
			return fmt.Errorf("添加 agent_status 列失败: %w", err)
		}
	}

	// 添加 agent_check_time 列
	var agentCheckTimeExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='agent_check_time'").Scan(&agentCheckTimeExists)
	if err == nil && agentCheckTimeExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN agent_check_time DATETIME"); err != nil {
			return fmt.Errorf("添加 agent_check_time 列失败: %w", err)
		}
	}

	// 添加 system_stats 列
	var systemStatsExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='system_stats'").Scan(&systemStatsExists)
	if err == nil && systemStatsExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN system_stats TEXT DEFAULT ''"); err != nil {
			return fmt.Errorf("添加 system_stats 列失败: %w", err)
		}
	}

	// 添加 environment_info 列
	var environmentInfoExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='environment_info'").Scan(&environmentInfoExists)
	if err == nil && environmentInfoExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN environment_info TEXT DEFAULT ''"); err != nil {
			return fmt.Errorf("添加 environment_info 列失败: %w", err)
		}
	}

	// 添加 agent_uptime 列
	var agentUptimeExists int
	err = DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info('slaves') WHERE name='agent_uptime'").Scan(&agentUptimeExists)
	if err == nil && agentUptimeExists == 0 {
		if _, err := DB.Exec("ALTER TABLE slaves ADD COLUMN agent_uptime INTEGER DEFAULT 0"); err != nil {
			return fmt.Errorf("添加 agent_uptime 列失败: %w", err)
		}
	}

	return nil
}

// createScriptVersionsTable 创建脚本版本表
func createScriptVersionsTable() error {
	table := `
	CREATE TABLE IF NOT EXISTS script_versions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		script_id INTEGER NOT NULL,
		version_number INTEGER NOT NULL,
		content TEXT NOT NULL,
		content_hash TEXT NOT NULL,
		change_summary TEXT,
		created_at DATETIME NOT NULL,
		FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE
	);`
	if _, err := DB.Exec(table); err != nil {
		return fmt.Errorf("创建 script_versions 表失败: %w", err)
	}
	return nil
}

// createIndexes 创建数据库索引
func createIndexes() error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_executions_script_id ON executions(script_id);`,
		`CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);`,
		`CREATE INDEX IF NOT EXISTS idx_executions_created_at ON executions(created_at DESC);`,
		`CREATE INDEX IF NOT EXISTS idx_script_files_script_id ON script_files(script_id);`,
		`CREATE INDEX IF NOT EXISTS idx_script_versions_script_id ON script_versions(script_id);`,
	}

	for _, sql := range indexes {
		if _, err := DB.Exec(sql); err != nil {
			return fmt.Errorf("执行索引创建失败 (%s): %w", sql, err)
		}
	}

	return nil
}

func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
