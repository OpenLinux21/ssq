package lotto

import "testing"

func TestSheets(t *testing.T) {
	mk := func(n int) []Ticket {
		ts := make([]Ticket, n)
		for i := range ts {
			// 用蓝球承载序号，便于验证分组顺序。
			ts[i] = Ticket{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: i + 1}
		}
		return ts
	}
	cases := []struct {
		name    string
		n       int
		wantLen []int // 每张票纸的注数
	}{
		{"零注", 0, nil},
		{"一注", 1, []int{1}},
		{"四注不足一张", 4, []int{4}},
		{"恰好一张", 5, []int{5}},
		{"六注跨两张", 6, []int{5, 1}},
		{"十注整两张", 10, []int{5, 5}},
		{"四十九注", 49, []int{5, 5, 5, 5, 5, 5, 5, 5, 5, 4}},
		{"满额五十注", 50, []int{5, 5, 5, 5, 5, 5, 5, 5, 5, 5}},
	}
	for _, c := range cases {
		got := Sheets(mk(c.n))
		if len(got) != len(c.wantLen) {
			t.Errorf("%s: 张数 %d，期望 %d", c.name, len(got), len(c.wantLen))
			continue
		}
		total := 0
		for i, sheet := range got {
			if len(sheet) != c.wantLen[i] {
				t.Errorf("%s: 第 %d 张 %d 注，期望 %d", c.name, i+1, len(sheet), c.wantLen[i])
			}
			total += len(sheet)
		}
		if total != c.n {
			t.Errorf("%s: 分组总注数 %d，期望 %d", c.name, total, c.n)
		}
	}

	// 验证分组保持原顺序：全局第 6 注应位于第 2 张的第 1 行。
	sheets := Sheets(mk(7))
	if sheets[1][0].Blue != 6 {
		t.Fatalf("分组顺序错误：第 2 张首注 Blue=%d，期望 6", sheets[1][0].Blue)
	}
}
