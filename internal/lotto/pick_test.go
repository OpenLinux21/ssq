package lotto

import (
	"errors"
	"strings"
	"testing"
)

// stubRandom 为可脚本化的 Randomizer 桩，按顺序返回预置结果。
type stubRandom struct {
	uniqueResults [][]int
	uniqueErr     error
	intnResults   []int
	intnErr       error
	uCalls        int
	iCalls        int
}

func (s *stubRandom) UniqueInts(max, count int) ([]int, error) {
	if s.uniqueErr != nil {
		return nil, s.uniqueErr
	}
	r := s.uniqueResults[s.uCalls%len(s.uniqueResults)]
	s.uCalls++
	return append([]int(nil), r...), nil
}

func (s *stubRandom) Intn(n int) (int, error) {
	if s.intnErr != nil {
		return 0, s.intnErr
	}
	r := s.intnResults[s.iCalls%len(s.intnResults)]
	s.iCalls++
	return r, nil
}

func TestParseReds(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   []int
		errSub string
	}{
		{"空格分隔", "01 02 03 04 05 06", []int{1, 2, 3, 4, 5, 6}, ""},
		{"混合分隔", "1,2，3、4 5 06", []int{1, 2, 3, 4, 5, 6}, ""},
		{"乱序输入", "33 22 11 9 3 1", []int{1, 3, 9, 11, 22, 33}, ""},
		{"多空格", "  7   8  9 10 11 12  ", []int{7, 8, 9, 10, 11, 12}, ""},
		{"数量不足", "1 2 3", nil, "6 个"},
		{"数量超限", "1 2 3 4 5 6 7", nil, "6 个"},
		{"超出范围", "01 02 03 04 05 34", nil, "超出范围"},
		{"红球重复", "01 02 03 04 05 05", nil, "重复"},
		{"非法字符", "01 02 03 04 05 0a", nil, "不是合法号码"},
		{"负数", "-1 02 03 04 05 06", nil, "不是合法号码"},
		{"三位数", "01 02 03 04 05 123", nil, "不是合法号码"},
		{"零值", "00 02 03 04 05 06", nil, "超出范围"},
		{"空输入", "   ", nil, "未输入任何号码"},
	}
	for _, c := range cases {
		got, err := ParseReds(c.input)
		if c.errSub != "" {
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Errorf("%s: 期望错误包含 %q，实际 %v", c.name, c.errSub, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: 意外报错: %v", c.name, err)
			continue
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: 结果数量错误: %v", c.name, got)
			continue
		}
		for i := range c.want {
			if got[i] != c.want[i] {
				t.Errorf("%s: 结果错误: %v（期望 %v）", c.name, got, c.want)
				break
			}
		}
	}
}

func TestParseBlue(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		want   int
		errSub string
	}{
		{"带前导零", "07", 7, ""},
		{"无前导零", "7", 7, ""},
		{"最小值", "1", 1, ""},
		{"最大值", "16", 16, ""},
		{"超出上限", "17", 0, "超出范围"},
		{"零", "0", 0, "超出范围"},
		{"多个号码", "1 2", 0, "1 个号码"},
		{"空输入", "", 0, "1 个号码"},
		{"非法字符", "x", 0, "不是合法号码"},
	}
	for _, c := range cases {
		got, err := ParseBlue(c.input)
		if c.errSub != "" {
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Errorf("%s: 期望错误包含 %q，实际 %v", c.name, c.errSub, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: 意外报错: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: 结果 %d，期望 %d", c.name, got, c.want)
		}
	}
}

func TestNewTicket(t *testing.T) {
	tk, err := NewTicket([]int{6, 5, 4, 3, 2, 1}, 9)
	if err != nil {
		t.Fatalf("合法输入报错: %v", err)
	}
	want := []int{1, 2, 3, 4, 5, 6}
	for i := range want {
		if tk.Reds[i] != want[i] {
			t.Fatalf("未升序排列: %v", tk.Reds)
		}
	}
	if tk.Blue != 9 {
		t.Fatalf("蓝球错误: %d", tk.Blue)
	}
	if _, err := NewTicket([]int{1, 2, 3, 4, 5, 34}, 9); err == nil {
		t.Fatal("非法红球应报错")
	}
	if _, err := NewTicket([]int{1, 2, 3, 4, 5, 6}, 17); err == nil {
		t.Fatal("非法蓝球应报错")
	}
}

func TestAutoPick(t *testing.T) {
	r := &stubRandom{
		uniqueResults: [][]int{{6, 33, 1, 15, 7, 22}},
		intnResults:   []int{4},
	}
	tk, err := AutoPick(r)
	if err != nil {
		t.Fatalf("AutoPick: %v", err)
	}
	want := []int{1, 6, 7, 15, 22, 33}
	for i := range want {
		if tk.Reds[i] != want[i] {
			t.Fatalf("红球错误: %v（期望 %v）", tk.Reds, want)
		}
	}
	if tk.Blue != 5 {
		t.Fatalf("蓝球错误: %d（期望 Intn(16)=4 + 1 = 5）", tk.Blue)
	}
	if err := ValidateTicket(tk); err != nil {
		t.Fatalf("机选结果未通过校验: %v", err)
	}
}

func TestAutoPickPropagatesErrors(t *testing.T) {
	r := &stubRandom{uniqueErr: errors.New("boom")}
	if _, err := AutoPick(r); err == nil || !strings.Contains(err.Error(), "机选红球失败") {
		t.Fatalf("红球错误应传播: %v", err)
	}
	r2 := &stubRandom{
		uniqueResults: [][]int{{1, 2, 3, 4, 5, 6}},
		intnErr:       errors.New("boom"),
	}
	if _, err := AutoPick(r2); err == nil || !strings.Contains(err.Error(), "机选蓝球失败") {
		t.Fatalf("蓝球错误应传播: %v", err)
	}
}
