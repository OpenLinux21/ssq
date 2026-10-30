package lotto

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Randomizer 为随机数供给接口，由 randx.Source 实现；
// 单元测试可注入桩实现以获得确定性的选号结果。
type Randomizer interface {
	// Intn 返回 [0, n) 内的无偏均匀随机整数。
	Intn(n int) (int, error)
	// UniqueInts 从 [1, max] 中抽取 count 个互不相同的整数。
	UniqueInts(max, count int) ([]int, error)
}

// NewTicket 由已解析的红球与蓝球构造一注号码：先校验，再按升序规范化。
func NewTicket(reds []int, blue int) (Ticket, error) {
	t := Ticket{Reds: append([]int(nil), reds...), Blue: blue}
	if err := ValidateTicket(t); err != nil {
		return Ticket{}, err
	}
	return SortTicket(t), nil
}

// ParseReds 解析一行红球输入：6 个号码，可用空格、英文逗号、
// 中文逗号、顿号、分号或制表符分隔；允许前导零（如 "07"）。
// 输入依次经过格式、数量、范围与重复校验，返回升序结果。
func ParseReds(line string) ([]int, error) {
	fields := splitFields(line)
	if len(fields) == 0 {
		return nil, errors.New("未输入任何号码")
	}
	reds := make([]int, 0, len(fields))
	for _, f := range fields {
		n, err := parseBall(f)
		if err != nil {
			return nil, fmt.Errorf("“%s”不是合法号码: %w", f, err)
		}
		reds = append(reds, n)
	}
	if err := ValidateReds(reds); err != nil {
		return nil, err
	}
	sort.Ints(reds)
	return reds, nil
}

// ParseBlue 解析一行蓝球输入：恰好 1 个号码。
func ParseBlue(line string) (int, error) {
	fields := splitFields(line)
	if len(fields) != 1 {
		return 0, fmt.Errorf("蓝球必须为 %d 个号码，实际输入 %d 个", BlueCount, len(fields))
	}
	n, err := parseBall(fields[0])
	if err != nil {
		return 0, fmt.Errorf("“%s”不是合法号码: %w", fields[0], err)
	}
	if err := ValidateBlue(n); err != nil {
		return 0, err
	}
	return n, nil
}

// parseBall 解析单个号码：必须是 1-2 位纯数字，禁止 "1a"、"+1"、"-1"、"123" 等。
func parseBall(s string) (int, error) {
	if s == "" {
		return 0, errors.New("存在空号码")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("包含非数字字符")
		}
	}
	if len(s) > 2 {
		return 0, errors.New("位数过多（最多 2 位）")
	}
	return strconv.Atoi(s)
}

// splitFields 将输入按常见分隔符切分为号码字段。
func splitFields(line string) []string {
	rep := strings.NewReplacer(
		",", " ", "，", " ", "、", " ", ";", " ", "；", " ", "\t", " ",
	)
	return strings.Fields(rep.Replace(line))
}

// AutoPick 机选一注号码：6 个不重复红球 + 1 个蓝球，红球升序。
// 内部结果会再次校验，确保始终返回符合规则的号码。
func AutoPick(r Randomizer) (Ticket, error) {
	reds, err := r.UniqueInts(RedMax, RedCount)
	if err != nil {
		return Ticket{}, fmt.Errorf("机选红球失败: %w", err)
	}
	sort.Ints(reds)
	blueIdx, err := r.Intn(BlueMax)
	if err != nil {
		return Ticket{}, fmt.Errorf("机选蓝球失败: %w", err)
	}
	t := Ticket{Reds: reds, Blue: blueIdx + BlueMin}
	if err := ValidateTicket(t); err != nil {
		return Ticket{}, fmt.Errorf("机选号码未通过校验（内部错误）: %w", err)
	}
	return t, nil
}
