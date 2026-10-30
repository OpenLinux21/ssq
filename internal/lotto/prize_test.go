package lotto

import "testing"

func TestJudge(t *testing.T) {
	cases := []struct {
		name    string
		redHits int
		blueHit bool
		special bool
		want    PrizeLevel
	}{
		{"6+1 一等奖", 6, true, false, PrizeFirst},
		{"6+1 特别规定期间仍一等奖", 6, true, true, PrizeFirst},
		{"6+0 二等奖", 6, false, false, PrizeSecond},
		{"5+1 三等奖", 5, true, false, PrizeThird},
		{"5+1 特别规定期间仍三等奖", 5, true, true, PrizeThird},
		{"5+0 四等奖", 5, false, false, PrizeFour},
		{"4+1 四等奖", 4, true, false, PrizeFour},
		{"4+0 五等奖", 4, false, false, PrizeFive},
		{"3+1 五等奖（不叠加福运奖）", 3, true, true, PrizeFive},
		{"3+0 特别规定期间中福运奖", 3, false, true, PrizeFuyun},
		{"3+0 未执行特别规定不中奖", 3, false, false, PrizeNone},
		{"2+1 六等奖", 2, true, false, PrizeSix},
		{"1+1 六等奖", 1, true, false, PrizeSix},
		{"0+1 六等奖", 0, true, false, PrizeSix},
		{"2+0 不中奖", 2, false, true, PrizeNone},
		{"1+0 不中奖", 1, false, false, PrizeNone},
		{"0+0 不中奖", 0, false, true, PrizeNone},
	}
	for _, c := range cases {
		if got := Judge(c.redHits, c.blueHit, c.special); got != c.want {
			t.Errorf("%s: Judge(%d, %v, %v) = %v，期望 %v",
				c.name, c.redHits, c.blueHit, c.special, got, c.want)
		}
	}
}

func TestCountRedHitsAndBlueHit(t *testing.T) {
	draw := DrawResult{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 7}

	hit3 := Ticket{Reds: []int{2, 4, 6, 10, 20, 30}, Blue: 7}
	if got := CountRedHits(hit3, draw); got != 3 {
		t.Fatalf("红球命中数期望 3，实际 %d", got)
	}
	if !BlueHit(hit3, draw) {
		t.Fatal("蓝球应命中")
	}

	hit6 := Ticket{Reds: []int{6, 5, 4, 3, 2, 1}, Blue: 8}
	if got := CountRedHits(hit6, draw); got != 6 {
		t.Fatalf("乱序红球命中数期望 6，实际 %d", got)
	}
	if BlueHit(hit6, draw) {
		t.Fatal("蓝球不应命中")
	}

	miss := Ticket{Reds: []int{28, 29, 30, 31, 32, 33}, Blue: 16}
	if got := CountRedHits(miss, draw); got != 0 {
		t.Fatalf("红球命中数期望 0，实际 %d", got)
	}
}

func TestPrizeLevelMetadata(t *testing.T) {
	fixed := map[PrizeLevel]int{
		PrizeThird: 3000,
		PrizeFour:  200,
		PrizeFive:  10,
		PrizeSix:   5,
		PrizeFuyun: 5,
	}
	for lvl, want := range fixed {
		got, ok := lvl.FixedAmount()
		if !ok || got != want {
			t.Errorf("%v.FixedAmount() = (%d, %v)，期望 (%d, true)", lvl, got, ok, want)
		}
		if lvl.IsFloating() {
			t.Errorf("%v 不应为浮动奖级", lvl)
		}
	}
	for _, lvl := range []PrizeLevel{PrizeFirst, PrizeSecond} {
		if _, ok := lvl.FixedAmount(); ok {
			t.Errorf("%v 应为浮动奖级", lvl)
		}
		if !lvl.IsFloating() {
			t.Errorf("%v 应为浮动奖级", lvl)
		}
	}
	if PrizeNone.String() != "未中奖" {
		t.Errorf("未中奖命名异常: %q", PrizeNone.String())
	}
	if PrizeFuyun.String() != "福运奖" {
		t.Errorf("福运奖命名异常: %q", PrizeFuyun.String())
	}
}

func TestJudgeFuyunScenarioWithRealTickets(t *testing.T) {
	draw := DrawResult{Reds: []int{1, 2, 3, 4, 5, 6}, Blue: 7}
	tk := Ticket{Reds: []int{1, 2, 3, 20, 21, 22}, Blue: 16} // 3 红 0 蓝

	if got := Judge(CountRedHits(tk, draw), BlueHit(tk, draw), false); got != PrizeNone {
		t.Fatalf("未执行特别规定时 3+0 应未中奖，实际 %v", got)
	}
	if got := Judge(CountRedHits(tk, draw), BlueHit(tk, draw), true); got != PrizeFuyun {
		t.Fatalf("执行特别规定时 3+0 应为福运奖，实际 %v", got)
	}
}
