package service

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

const maxCSVScanTokenSize = 8 * 1024 * 1024

func newCSVScanner(file *os.File) *bufio.Scanner {
	scanner := bufio.NewScanner(file)
	buffer := make([]byte, 0, 64*1024)
	scanner.Buffer(buffer, maxCSVScanTokenSize)
	return scanner
}

// SplitCSV 将 CSV 文件均匀拆分为 partCount 份
// filePath: 原始 CSV 文件路径
// partCount: 分片数量
// hasHeader: 是否有表头（第一行为表头，每个分片都保留）
// outputDir: 分片文件输出目录
// baseName: 原始文件名（不含路径）
// 返回: 分片文件路径列表（按序号 0,1,2... 排列）
func SplitCSV(filePath string, partCount int, hasHeader bool, outputDir string, baseName string) ([]string, error) {
	if partCount <= 0 {
		return nil, fmt.Errorf("分片数量必须大于0")
	}

	// 打开原始文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开CSV文件失败: %w", err)
	}
	defer file.Close()

	// 第一遍扫描：统计总行数
	totalRows := 0
	scanner := newCSVScanner(file)
	for scanner.Scan() {
		totalRows++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("扫描CSV文件失败: %w", err)
	}

	// 计算数据行数（不含表头）
	dataRows := totalRows
	if hasHeader && totalRows > 0 {
		dataRows = totalRows - 1
	}

	if dataRows == 0 {
		return nil, fmt.Errorf("CSV文件没有数据行")
	}

	// 重新打开文件进行第二遍扫描
	file.Seek(0, 0)
	scanner = newCSVScanner(file)

	// 读取表头
	var header string
	if hasHeader && scanner.Scan() {
		header = scanner.Text()
	}

	// 计算每个分片的行数
	// 均分逻辑：base = dataRows / partCount, remainder = dataRows % partCount
	// 前 remainder 个分片各 base+1 行，其余各 base 行
	baseRows := dataRows / partCount
	remainder := dataRows % partCount

	// 创建分片文件
	partFiles := make([]*os.File, partCount)
	partPaths := make([]string, partCount)
	partWriters := make([]*bufio.Writer, partCount)

	for i := 0; i < partCount; i++ {
		partName := fmt.Sprintf("%s_part%d.csv", baseName, i)
		partPath := filepath.Join(outputDir, partName)
		partPaths[i] = partPath

		f, err := os.Create(partPath)
		if err != nil {
			// 清理已创建的文件
			for j := 0; j < i; j++ {
				partFiles[j].Close()
				os.Remove(partPaths[j])
			}
			return nil, fmt.Errorf("创建分片文件失败: %w", err)
		}
		partFiles[i] = f
		partWriters[i] = bufio.NewWriter(f)

		// 写入表头
		if hasHeader && header != "" {
			partWriters[i].WriteString(header)
			partWriters[i].WriteByte('\n')
		}
	}

	// 分配数据行到各分片
	currentPart := 0
	rowsInCurrentPart := 0
	targetRowsForPart := baseRows
	if remainder > 0 {
		targetRowsForPart = baseRows + 1
	}

	for scanner.Scan() {
		line := scanner.Text()

		// 写入当前分片
		partWriters[currentPart].WriteString(line)
		partWriters[currentPart].WriteByte('\n')
		rowsInCurrentPart++

		// 检查是否需要切换到下一个分片
		if rowsInCurrentPart >= targetRowsForPart && currentPart < partCount-1 {
			currentPart++
			rowsInCurrentPart = 0
			if currentPart < remainder {
				targetRowsForPart = baseRows + 1
			} else {
				targetRowsForPart = baseRows
			}
		}
	}

	if err := scanner.Err(); err != nil {
		// 清理文件
		for i := 0; i < partCount; i++ {
			if partFiles[i] != nil {
				partFiles[i].Close()
				os.Remove(partPaths[i])
			}
		}
		return nil, fmt.Errorf("读取CSV数据失败: %w", err)
	}

	// 刷新并关闭所有分片文件
	for i := 0; i < partCount; i++ {
		if err := partWriters[i].Flush(); err != nil {
			return nil, fmt.Errorf("刷新分片文件失败: %w", err)
		}
		if err := partFiles[i].Close(); err != nil {
			return nil, fmt.Errorf("关闭分片文件失败: %w", err)
		}
	}

	return partPaths, nil
}
