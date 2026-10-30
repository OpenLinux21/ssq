package tui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"
)

// 预定义输入错误，便于上层区分处理。
var (
	// ErrQuit 表示用户主动退出。
	ErrQuit = errors.New("用户退出")

	// ErrEOF 表示标准输入已结束（EOF）。
	ErrEOF = errors.New("标准输入已结束")
)

// Prompt 输出提示标签并读取一行输入（去掉行尾换行符）。
// io.EOF 归一为 ErrEOF；无换行符的最后一行仍作为有效输入返回。
func Prompt(r *bufio.Reader, label string) (string, error) {
	fmt.Fprint(out, label)
	line, err := r.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				return trimmed, nil
			}
			return "", ErrEOF
		}
		return "", fmt.Errorf("读取输入失败: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}
