package lotto

import (
	"errors"
	"strings"
	"testing"
)

// TestNewDraw 验证开奖号码的生成与字段映射（通过桩随机源获得确定结果）。
func TestNewDraw(t *testing.T) {
	r := &stubRandom{
		uniqueResults: [][]int{{6, 33, 1, 15, 7, 22}},
		intnResults:   []int{15},
	}
	d, err := NewDraw(r)
	if err != nil {
		t.Fatalf("NewDraw: %v", err)
	}
	want := []int{1, 6, 7, 15, 22, 33}
	for i := range want {
		if d.Reds[i] != want[i] {
			t.Fatalf("开奖红球错误: %v（期望 %v）", d.Reds, want)
		}
	}
	if d.Blue != 16 {
		t.Fatalf("开奖蓝球错误: %d（期望 Intn(16)=15 + 1 = 16）", d.Blue)
	}
}

func TestNewDrawPropagatesError(t *testing.T) {
	r := &stubRandom{uniqueErr: errors.New("boom")}
	if _, err := NewDraw(r); err == nil || !strings.Contains(err.Error(), "生成开奖号码失败") {
		t.Fatalf("错误应传播: %v", err)
	}
}

func TestJudgeAllAndCalcStatsWithFloating(t *testing.T) {
	draw := DrawResult{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 7}
	tickets := []Ticket{
		{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 7},        // 一等奖（浮动）
		{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 8},        // 二等奖（浮动）
		{Reds: []int{1, 2, 3, 20, 21, 22}, Blue: 7},     // 3+1 五等奖（固定 10 元）
		{Reds: []int{28, 29, 30, 31, 32, 33}, Blue: 16}, // 未中奖
	}

	r := JudgeAll(tickets, draw, false)
	wantLevels := []PrizeLevel{PrizeFirst, PrizeSecond, PrizeFive, PrizeNone}
	for i, want := range wantLevels {
		if r.Tickets[i].Level != want {
			t.Fatalf("第 %d 注期望 %v，实际 %v", i+1, want, r.Tickets[i].Level)
		}
	}

	s := CalcStats(r)
	if s.TicketCount != 4 || s.TotalCost != 8 {
		t.Fatalf("注数/成本统计错误: %+v", s)
	}
	if s.WinCount != 3 {
		t.Fatalf("中奖注数期望 3，实际 %d", s.WinCount)
	}
	if s.FixedTotal != 10 {
		t.Fatalf("固定奖金总额期望 10，实际 %d", s.FixedTotal)
	}
	if s.FloatingCount != 2 {
		t.Fatalf("浮动奖金注数期望 2，实际 %d", s.FloatingCount)
	}
	if s.NetDeterminable() != 2 {
		t.Fatalf("可确定净收益期望 10-8=2，实际 %d", s.NetDeterminable())
	}
	if s.NetFinalKnown() {
		t.Fatal("存在浮动奖金注时，最终净收益不应可准确计算")
	}
}

func TestCalcStatsAllKnown(t *testing.T) {
	draw := DrawResult{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 7}
	tickets := []Ticket{
		{Reds: []int{1, 2, 3, 20, 21, 22}, Blue: 7},     // 五等奖 10 元
		{Reds: []int{28, 29, 30, 31, 32, 33}, Blue: 16}, // 未中奖
	}
	r := JudgeAll(tickets, draw, false)
	s := CalcStats(r)
	if !s.NetFinalKnown() {
		t.Fatal("无浮动奖金注时，最终净收益应可准确计算")
	}
	if s.NetDeterminable() != 6 {
		t.Fatalf("可确定净收益期望 10-4=6，实际 %d", s.NetDeterminable())
	}
}

func TestCalcStatsEmpty(t *testing.T) {
	r := JudgeAll(nil, DrawResult{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 7}, false)
	s := CalcStats(r)
	if s.TicketCount != 0 || s.TotalCost != 0 || s.NetDeterminable() != 0 || !s.NetFinalKnown() {
		t.Fatalf("空投注统计异常: %+v", s)
	}
}
