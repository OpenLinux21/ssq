package lotto

import (
	"strings"
	"testing"
)

func TestValidateReds(t *testing.T) {
	cases := []struct {
		name string
		reds []int
		ok   bool
	}{
		{"标准六红", []int{1, 2, 3, 4, 5, 6}, true},
		{"乱序六红", []int{33, 1, 16, 7, 2, 9}, true},
		{"边界号码", []int{1, 33, 2, 3, 4, 5}, true},
		{"数量不足", []int{1, 2, 3, 4, 5}, false},
		{"数量超限", []int{1, 2, 3, 4, 5, 6, 7}, false},
		{"低于下限", []int{0, 1, 2, 3, 4, 5}, false},
		{"高于上限", []int{1, 2, 3, 4, 5, 34}, false},
		{"红球重复", []int{7, 7, 8, 9, 10, 11}, false},
		{"空切片", nil, false},
	}
	for _, c := range cases {
		err := ValidateReds(c.reds)
		if c.ok && err != nil {
			t.Errorf("%s: 意外报错: %v", c.name, err)
		}
		if !c.ok && err == nil {
			t.Errorf("%s: 期望报错但没有", c.name)
		}
	}
}

func TestValidateRedsErrorMessages(t *testing.T) {
	if err := ValidateReds([]int{1, 2, 3}); err == nil || !strings.Contains(err.Error(), "6 个") {
		t.Fatalf("数量错误提示不当: %v", err)
	}
	if err := ValidateReds([]int{1, 2, 3, 4, 5, 34}); err == nil || !strings.Contains(err.Error(), "超出范围") {
		t.Fatalf("范围错误提示不当: %v", err)
	}
	if err := ValidateReds([]int{1, 1, 2, 3, 4, 5}); err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("重复错误提示不当: %v", err)
	}
}

func TestValidateBlue(t *testing.T) {
	for _, ok := range []int{1, 8, 16} {
		if err := ValidateBlue(ok); err != nil {
			t.Errorf("蓝球 %d 应合法: %v", ok, err)
		}
	}
	for _, bad := range []int{0, -1, 17, 99} {
		if err := ValidateBlue(bad); err == nil {
			t.Errorf("蓝球 %d 应非法", bad)
		}
	}
}

func TestValidateTicket(t *testing.T) {
	good := Ticket{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 7}
	if err := ValidateTicket(good); err != nil {
		t.Fatalf("合法号码被误判: %v", err)
	}
	badReds := Ticket{Reds: []int{1, 2, 3, 4, 5}, Blue: 7}
	if err := ValidateTicket(badReds); err == nil {
		t.Fatal("红球数量错误应报错")
	}
	badBlue := Ticket{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 17}
	if err := ValidateTicket(badBlue); err == nil {
		t.Fatal("蓝球越界应报错")
	}
}

func TestSortTicketDoesNotMutateOriginal(t *testing.T) {
	in := Ticket{Reds: []int{9, 1, 5}, Blue: 3}
	got := SortTicket(in)
	want := []int{1, 5, 9}
	for i := range want {
		if got.Reds[i] != want[i] {
			t.Fatalf("排序结果错误: %v", got.Reds)
		}
	}
	if in.Reds[0] != 9 {
		t.Fatal("SortTicket 不应修改原切片")
	}
	if got.Blue != 3 {
		t.Fatalf("蓝球应保持不变: %d", got.Blue)
	}
}
