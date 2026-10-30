package lotto

// TicketResult 为单注对奖结果。
type TicketResult struct {
	Ticket  Ticket
	RedHits int  // 红球命中个数
	BlueHit bool // 蓝球是否命中
	Level   PrizeLevel
}

// Result 为一期完整的对奖结果。
type Result struct {
	Draw          DrawResult
	Tickets       []TicketResult
	SpecialActive bool // 是否执行特别规定（福运奖生效）
}

// JudgeAll 将每注投注与开奖结果比对并判定奖级。
func JudgeAll(tickets []Ticket, draw DrawResult, specialActive bool) Result {
	r := Result{Draw: draw, SpecialActive: specialActive}
	r.Tickets = make([]TicketResult, 0, len(tickets))
	for _, t := range tickets {
		tr := TicketResult{
			Ticket:  t,
			RedHits: CountRedHits(t, draw),
			BlueHit: BlueHit(t, draw),
		}
		tr.Level = Judge(tr.RedHits, tr.BlueHit, specialActive)
		r.Tickets = append(r.Tickets, tr)
	}
	return r
}

// Stats 为一期投注的统计汇总（金额单位：元）。
type Stats struct {
	TicketCount   int // 总注数
	TotalCost     int // 总投注成本 = TicketPrice × 注数
	WinCount      int // 中奖注数（任意奖级）
	FixedTotal    int // 固定奖金总额
	FloatingCount int // 浮动奖金注数（一、二等奖注数，金额待官方核定）
}

// CalcStats 汇总统计一期结果。
func CalcStats(r Result) Stats {
	s := Stats{
		TicketCount: len(r.Tickets),
		TotalCost:   len(r.Tickets) * TicketPrice,
	}
	for _, tr := range r.Tickets {
		if tr.Level == PrizeNone {
			continue
		}
		s.WinCount++
		if amt, fixed := tr.Level.FixedAmount(); fixed {
			s.FixedTotal += amt
		} else {
			s.FloatingCount++
		}
	}
	return s
}

// NetDeterminable 返回“可确定的净收益” = 固定奖金总额 − 总投注成本。
// 当存在浮动奖金注（FloatingCount > 0）时，最终净收益无法据此准确计算，
// 应结合 NetFinalKnown 一并展示说明。
func (s Stats) NetDeterminable() int {
	return s.FixedTotal - s.TotalCost
}

// NetFinalKnown 报告最终净收益是否可以准确计算
// （即不存在任何浮动奖金注）。
func (s Stats) NetFinalKnown() bool {
	return s.FloatingCount == 0
}
