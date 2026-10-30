// Package lotto 实现双色球的领域逻辑：号码验证、手动解析与机选、
// 开奖号码生成、中奖等级判定与奖金统计。
//
// 本包不依赖终端与操作系统细节，核心逻辑全部可通过单元测试覆盖。
//
// 规则依据：《中国福利彩票双色球游戏规则》（中国福彩网 2026-01-29 发布，
// 自第 2026014 期起执行）。若官方规则再调整，以官方公告为准。
package lotto

// 双色球基本常量。
const (
	RedMin   = 1  // 红球最小号码
	RedMax   = 33 // 红球最大号码
	RedCount = 6  // 每注红球个数

	BlueMin   = 1  // 蓝球最小号码
	BlueMax   = 16 // 蓝球最大号码
	BlueCount = 1  // 每注蓝球个数

	TicketPrice = 2 // 单注投注金额（元）
	MaxTickets  = 5 // 单次模拟最多投注注数
)

// Ticket 一注双色球投注号码。
type Ticket struct {
	Reds []int // 6 个红球，升序、不重复，范围 [RedMin, RedMax]
	Blue int   // 1 个蓝球，范围 [BlueMin, BlueMax]
}

// DrawResult 一期模拟开奖结果。
type DrawResult struct {
	Reds []int // 6 个红球，升序、不重复
	Blue int   // 1 个蓝球
}
