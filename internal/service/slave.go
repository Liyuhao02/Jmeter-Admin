package service

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"jmeter-admin/internal/database"
	"jmeter-admin/internal/model"
)

// ListSlaves 查询所有slave列表
func ListSlaves() ([]model.Slave, error) {
	rows, err := database.DB.Query("SELECT id, name, host, port, status, last_check_time, created_at FROM slaves ORDER BY id DESC")
	if err != nil {
		return nil, fmt.Errorf("查询slave列表失败: %w", err)
	}
	defer rows.Close()

	var slaves []model.Slave
	for rows.Next() {
		var slave model.Slave
		var lastCheckTime sql.NullString
		if err := rows.Scan(&slave.ID, &slave.Name, &slave.Host, &slave.Port, &slave.Status, &lastCheckTime, &slave.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描slave数据失败: %w", err)
		}
		if lastCheckTime.Valid {
			slave.LastCheckTime = lastCheckTime.String
		}
		slaves = append(slaves, slave)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历slave数据失败: %w", err)
	}

	return slaves, nil
}

// CreateSlave 创建slave
func CreateSlave(name, host string, port int) (*model.Slave, error) {
	now := time.Now().Format("2006-01-02 15:04:05")
	result, err := database.DB.Exec(
		"INSERT INTO slaves (name, host, port, status, created_at) VALUES (?, ?, ?, ?, ?)",
		name, host, port, "offline", now,
	)
	if err != nil {
		return nil, fmt.Errorf("创建slave失败: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("获取slave ID失败: %w", err)
	}

	slave := &model.Slave{
		ID:        id,
		Name:      name,
		Host:      host,
		Port:      port,
		Status:    "offline",
		CreatedAt: now,
	}

	return slave, nil
}

// UpdateSlave 更新slave
func UpdateSlave(id int64, name, host string, port int) error {
	result, err := database.DB.Exec(
		"UPDATE slaves SET name = ?, host = ?, port = ? WHERE id = ?",
		name, host, port, id,
	)
	if err != nil {
		return fmt.Errorf("更新slave失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slave不存在")
	}

	return nil
}

// DeleteSlave 删除slave
func DeleteSlave(id int64) error {
	result, err := database.DB.Exec("DELETE FROM slaves WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("删除slave失败: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("获取影响行数失败: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("slave不存在")
	}

	return nil
}

// CheckSlave 检测slave连通性
func CheckSlave(id int64) (bool, error) {
	// 获取slave信息
	var slave model.Slave
	var lastCheckTime sql.NullString
	err := database.DB.QueryRow(
		"SELECT id, name, host, port, status, last_check_time, created_at FROM slaves WHERE id = ?",
		id,
	).Scan(&slave.ID, &slave.Name, &slave.Host, &slave.Port, &slave.Status, &lastCheckTime, &slave.CreatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("slave不存在")
		}
		return false, fmt.Errorf("查询slave失败: %w", err)
	}
	if lastCheckTime.Valid {
		slave.LastCheckTime = lastCheckTime.String
	}

	// 检测连通性
	address := fmt.Sprintf("%s:%d", slave.Host, slave.Port)
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)

	var isOnline bool
	if err != nil {
		isOnline = false
	} else {
		isOnline = true
		conn.Close()
	}

	// 更新状态和最后检测时间
	status := "offline"
	if isOnline {
		status = "online"
	}
	now := time.Now().Format("2006-01-02 15:04:05")

	_, err = database.DB.Exec("UPDATE slaves SET status = ?, last_check_time = ? WHERE id = ?", status, now, id)
	if err != nil {
		return isOnline, fmt.Errorf("更新slave状态失败: %w", err)
	}

	return isOnline, nil
}

// StartHeartbeat 启动定时心跳检测
func StartHeartbeat(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		// 启动时立即执行一次
		checkAllSlaves()
		for range ticker.C {
			checkAllSlaves()
		}
	}()
}

func checkAllSlaves() {
	slaves, err := ListSlaves()
	if err != nil {
		log.Printf("心跳检测: 获取 Slave 列表失败: %v", err)
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 限制并发数为10

	for _, slave := range slaves {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量

		go func(s model.Slave) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			// 检测连通性
			address := fmt.Sprintf("%s:%d", s.Host, s.Port)
			conn, err := net.DialTimeout("tcp", address, 3*time.Second)

			var isOnline bool
			if err != nil {
				isOnline = false
			} else {
				isOnline = true
				conn.Close()
			}

			// 更新状态和最后检测时间
			status := "offline"
			if isOnline {
				status = "online"
			}
			now := time.Now().Format("2006-01-02 15:04:05")

			_, dbErr := database.DB.Exec("UPDATE slaves SET status = ?, last_check_time = ? WHERE id = ?", status, now, s.ID)
			if dbErr != nil {
				log.Printf("心跳检测: 更新 Slave %s 状态失败: %v", s.Name, dbErr)
			}

			log.Printf("心跳检测: %s (%s:%d) - %s", s.Name, s.Host, s.Port, map[bool]string{true: "online", false: "offline"}[isOnline])
		}(slave)
	}

	wg.Wait()
}
