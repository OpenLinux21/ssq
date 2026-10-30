package lotto

import (
	"fmt"
	"sort"
)

// ValidateReds 校验红球切片的个数、取值范围与重复性（不要求有序）。
func ValidateReds(reds []int) error {
	if len(reds) != RedCount {
		return fmt.Errorf("红球必须恰好 %d 个，实际输入 %d 个", RedCount, len(reds))
	}
	seen := make(map[int]bool, RedCount)
	for _, r := range reds {
		if r < RedMin || r > RedMax {
			return fmt.Errorf("红球 %02d 超出范围 %02d-%02d", r, RedMin, RedMax)
		}
		if seen[r] {
			return fmt.Errorf("红球 %02d 重复", r)
		}
		seen[r] = true
	}
	return nil
}

// ValidateBlue 校验蓝球号码的取值范围。
func ValidateBlue(blue int) error {
	if blue < BlueMin || blue > BlueMax {
		return fmt.Errorf("蓝球 %02d 超出范围 %02d-%02d", blue, BlueMin, BlueMax)
	}
	return nil
}

// ValidateTicket 校验一注号码的完整性。
func ValidateTicket(t Ticket) error {
	if err := ValidateReds(t.Reds); err != nil {
		return err
	}
	return ValidateBlue(t.Blue)
}

// SortTicket 返回红球升序排列的副本，不修改原切片。
func SortTicket(t Ticket) Ticket {
	reds := append([]int(nil), t.Reds...)
	sort.Ints(reds)
	return Ticket{Reds: reds, Blue: t.Blue}
}
