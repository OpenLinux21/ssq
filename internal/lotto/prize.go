package lotto

// PrizeLevel 表示中奖奖级，数值越大奖级越高。
type PrizeLevel int

// 奖级定义。
const (
	PrizeNone   PrizeLevel = iota // 未中奖
	PrizeFuyun                    // 福运奖（固定 5 元，仅“执行特别规定期间”有效）
	PrizeSix                      // 六等奖（固定 5 元）
	PrizeFive                     // 五等奖（固定 10 元）
	PrizeFour                     // 四等奖（固定 200 元）
	PrizeThird                    // 三等奖（固定 3000 元）
	PrizeSecond                   // 二等奖（浮动，待官方核定）
	PrizeFirst                    // 一等奖（浮动，待官方核定）
)

// 固定奖金金额（元）。
// 规则依据：《中国福利彩票双色球游戏规则》第十七、十八条
// （中国福彩网 2026-01-29 发布，自第 2026014 期起执行）。
const (
	FixedThird = 3000 // 三等奖
	FixedFour  = 200  // 四等奖
	FixedFive  = 10   // 五等奖
	FixedSix   = 5    // 六等奖
	FixedFuyun = 5    // 福运奖
)

// String 返回奖级中文名。
func (p PrizeLevel) String() string {
	switch p {
	case PrizeFirst:
		return "一等奖"
	case PrizeSecond:
		return "二等奖"
	case PrizeThird:
		return "三等奖"
	case PrizeFour:
		return "四等奖"
	case PrizeFive:
		return "五等奖"
	case PrizeSix:
		return "六等奖"
	case PrizeFuyun:
		return "福运奖"
	default:
		return "未中奖"
	}
}

// IsFloating 报告是否为浮动奖金奖级（一等奖、二等奖）。
// 浮动奖金取决于当期奖池与销售额等官方数据，本模拟器一律显示“待官方核定”，
// 不编造具体金额。
func (p PrizeLevel) IsFloating() bool {
	return p == PrizeFirst || p == PrizeSecond
}

// FixedAmount 返回固定奖金金额（元）；浮动奖级第二个返回值为 false。
func (p PrizeLevel) FixedAmount() (int, bool) {
	switch p {
	case PrizeThird:
		return FixedThird, true
	case PrizeFour:
		return FixedFour, true
	case PrizeFive:
		return FixedFive, true
	case PrizeSix:
		return FixedSix, true
	case PrizeFuyun:
		return FixedFuyun, true
	default:
		return 0, false
	}
}

// CountRedHits 统计投注红球与开奖红球的命中个数（顺序不限）。
func CountRedHits(t Ticket, d DrawResult) int {
	set := make(map[int]bool, len(d.Reds))
	for _, r := range d.Reds {
		set[r] = true
	}
	n := 0
	for _, r := range t.Reds {
		if set[r] {
			n++
		}
	}
	return n
}

// BlueHit 报告蓝球是否命中。
func BlueHit(t Ticket, d DrawResult) bool {
	return t.Blue == d.Blue
}

// Judge 依据红球命中数与蓝球命中情况判定奖级（单注唯一奖级）。
//
// 规则依据（第二十八条、第二十九条）：
//   - 一等奖：6 红 + 1 蓝全部相同；
//   - 二等奖：6 个红球相同；
//   - 三等奖：任意 5 红 + 1 蓝；
//   - 四等奖：任意 5 红，或 4 红 + 1 蓝；
//   - 五等奖：任意 4 红，或 3 红 + 1 蓝；
//   - 六等奖：命中 1 个蓝球（红球不限）；
//   - 福运奖：执行特别规定期间，任意 3 个红球且未中蓝球（3+0）；
//   - 每注只兑付所中最高奖级，不能兼中兼得。
//
// 福运奖启用条件（第十八条）：当期奖池资金 ≥15 亿元时开始执行特别规定，
// 执行后直至某期开奖后奖池跌破 3 亿元时停止。该条件取决于官方奖池数据；
// 本程序为本地模拟器，不连接官方系统、无法获知真实奖池，因此
// specialActive 固定传入 false（未执行特别规定），也不向用户提供开关。
func Judge(redHits int, blueHit bool, specialActive bool) PrizeLevel {
	switch {
	case redHits == 6 && blueHit:
		return PrizeFirst
	case redHits == 6:
		return PrizeSecond
	case redHits == 5 && blueHit:
		return PrizeThird
	case redHits == 5, redHits == 4 && blueHit:
		return PrizeFour
	case redHits == 4, redHits == 3 && blueHit:
		return PrizeFive
	case redHits == 3 && specialActive:
		return PrizeFuyun // 3 红 0 蓝，仅执行特别规定期间
	case blueHit:
		return PrizeSix // 0/1/2 红 + 命中蓝球
	default:
		return PrizeNone
	}
}
