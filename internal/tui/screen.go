package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"ssq/internal/lotto"
)

// Version 为程序版本号。
const Version = "1.0.0"

// pageWidth 为界面内容宽度（终端列数近似值）。
const pageWidth = 56

// out 为统一输出口，便于测试替换。
var out = os.Stdout

// ───────────────────────────── 基础版式 ─────────────────────────────

// repeatLine 生成长度为 n 的水平分隔线。
func repeatLine(n int) string { return strings.Repeat("─", n) }

// Section 输出居中的小节标题，如“──── 对奖明细 ────”。
func Section(title string) {
	side := (pageWidth - displayWidth(title) - 2) / 2
	if side < 2 {
		side = 2
	}
	fmt.Fprintln(out, Dim(repeatLine(side)+" "+title+" "+repeatLine(side)))
}

// BoxLine 生成“│ content │”形式的边框行（按显示宽度居中）。
func BoxLine(content string) string {
	inner := pageWidth - 2
	cw := displayWidth(content)
	if cw > inner {
		cw = inner
	}
	left := (inner - cw) / 2
	right := inner - cw - left
	return "│" + strings.Repeat(" ", left) + content + strings.Repeat(" ", right) + "│"
}

// Box 输出一个居中内容的边框块。
func Box(lines ...string) {
	fmt.Fprintln(out, "┌"+repeatLine(pageWidth-2)+"┐")
	for _, l := range lines {
		fmt.Fprintln(out, BoxLine(l))
	}
	fmt.Fprintln(out, "└"+repeatLine(pageWidth-2)+"┘")
}

// kv 输出“标签：值”形式的信息行，标签列对齐；label 为空时输出纯补充行。
func kv(label, value string) {
	if label == "" {
		fmt.Fprintln(out, strings.Repeat(" ", 13)+value)
		return
	}
	fmt.Fprintln(out, "  "+Cyan(padRight(label+"：", 11))+value)
}

// Warn 输出黄色警告。
func Warn(msg string) { fmt.Fprintln(out, Yellow("⚠ ")+msg) }

// Info 输出青色提示。
func Info(msg string) { fmt.Fprintln(out, Cyan("ℹ ")+msg) }

// ───────────────────────────── 欢迎与菜单 ─────────────────────────────

// Welcome 输出欢迎界面：标题、版本、随机源状态、规则依据与免责声明。
func Welcome(devPath string) {
	fmt.Fprintln(out)
	Box(Bold(Cyan("双色球 · 本地模拟器")), Dim("SSQ Local Simulator · Linux TUI"))
	kv("程序版本", "v"+Version)
	kv("随机源", devPath+"（内核 CSPRNG · 已就绪）")
	kv("随机算法", "拒绝采样（无偏）· 不使用 math/rand")
	kv("回退策略", "阻塞超时 10s 后切换 /dev/urandom")
	kv("规则依据", "《中国福利彩票双色球游戏规则》")
	kv("", Dim("中国福彩网 2026-01-29 发布，自第 2026014 期起执行"))
	fmt.Fprintln(out, Dim(repeatLine(pageWidth)))
	fmt.Fprintln(out, Yellow("免责声明")+"　本程序为纯本地随机模拟：")
	fmt.Fprintln(out, "  · 不连接福利彩票官方系统，不读取、不伪造真实开奖号码")
	fmt.Fprintln(out, "  · 不能预测真实开奖；浮动奖金一律显示“待官方核定”")
	fmt.Fprintln(out)
}

// Menu 输出选号主菜单。
func Menu(ticketCount int) {
	Section(fmt.Sprintf("选号（已选 %d/%d 注）", ticketCount, lotto.MaxTickets))
	fmt.Fprintln(out, "  "+Green("[1]")+" 手动选号（录入一注）")
	fmt.Fprintln(out, "  "+Green("[2]")+" 自动机选一注")
	fmt.Fprintln(out, "  "+Green("[3]")+" 自动机选补满剩余全部")
	fmt.Fprintln(out, "  "+Green("[4]")+" 清空已选号码")
	fmt.Fprintln(out, "  "+Green("[5]")+" 确认投注并模拟开奖")
	fmt.Fprintln(out, "  "+Green("[0]")+" 退出程序")
	if ticketCount == 0 {
		fmt.Fprintln(out, Dim("  提示：先用 1/2/3 选号（每注 6 红 + 1 蓝），再用 5 确认。"))
	}
	fmt.Fprintln(out)
}

// ───────────────────────────── 号码展示 ─────────────────────────────

// ShowTickets 以芯片样式展示一组投注号码。
func ShowTickets(tickets []lotto.Ticket) {
	for i, t := range tickets {
		var b strings.Builder
		for _, r := range t.Reds {
			b.WriteString(RedBall(r))
			b.WriteString(" ")
		}
		b.WriteString(Dim("┃"))
		fmt.Fprintln(out, "  "+padRight(fmt.Sprintf("第 %d 注", i+1), 9)+b.String()+" "+BlueBall(t.Blue))
	}
}

// BallText 以着色两位数字渲染一注号码：“01 02 … 33 + 09”，总宽 22 列。
func BallText(t lotto.Ticket) string {
	parts := make([]string, 0, len(t.Reds))
	for _, r := range t.Reds {
		parts = append(parts, Red(fmt.Sprintf("%02d", r)))
	}
	return strings.Join(parts, " ") + Dim(" +") + " " + Blue(fmt.Sprintf("%02d", t.Blue))
}

// ConfirmView 展示全部投注号码与成本信息，用于确认阶段。
func ConfirmView(tickets []lotto.Ticket) {
	Section("投注确认")
	ShowTickets(tickets)
	kv("投注注数", fmt.Sprintf("%d / %d 注", len(tickets), lotto.MaxTickets))
	kv("投注成本", fmt.Sprintf("%d 元（%d 元/注）", len(tickets)*lotto.TicketPrice, lotto.TicketPrice))
}

// ───────────────────────────── 统计报告 ─────────────────────────────

// Report 输出开奖号码、对奖明细与统计汇总。
func Report(r lotto.Result, s lotto.Stats) {
	fmt.Fprintln(out)
	Section("模拟开奖号码")
	var b strings.Builder
	b.WriteString("  红球 ")
	for _, red := range r.Draw.Reds {
		b.WriteString(RedBall(red))
		b.WriteString(" ")
	}
	b.WriteString(Dim("┃") + " 蓝球 " + BlueBall(r.Draw.Blue))
	fmt.Fprintln(out, b.String())
	fmt.Fprintln(out)

	Section("对奖明细")
	header := "  " + padRight("注", 3) + padRight("投注号码", 24) +
		padLeft("红中", 5) + padLeft("蓝中", 5) + " " +
		padRight("奖级", 8) + padRight("单注奖金", 12)
	fmt.Fprintln(out, Bold(header))
	fmt.Fprintln(out, Dim("  "+repeatLine(pageWidth-2)))

	for i, tr := range r.Tickets {
		blueMark := Dim("✗")
		if tr.BlueHit {
			blueMark = Green(padLeft("✓", 5))
		} else {
			blueMark = Dim(padLeft("✗", 5))
		}
		level := Green(padRight(tr.Level.String(), 8))
		if tr.Level == lotto.PrizeNone {
			level = Dim(padRight("未中奖", 8))
		}
		money := Dim(padRight("—", 12))
		if amt, fixed := tr.Level.FixedAmount(); fixed {
			money = Green(padRight(fmt.Sprintf("%d 元", amt), 12))
		} else if tr.Level.IsFloating() {
			money = Yellow(padRight("待官方核定", 12))
		}
		fmt.Fprintln(out, "  "+padRight(strconv.Itoa(i+1), 3)+
			BallText(tr.Ticket)+"  "+padLeft(strconv.Itoa(tr.RedHits), 5)+blueMark+" "+level+money)
	}
	fmt.Fprintln(out)

	Section("统计汇总")
	kv("总投注注数", fmt.Sprintf("%d 注", s.TicketCount))
	kv("总投注成本", fmt.Sprintf("%d 元", s.TotalCost))
	kv("中奖注数", fmt.Sprintf("%d 注", s.WinCount))
	kv("固定奖金总额", fmt.Sprintf("%d 元", s.FixedTotal))
	if s.FloatingCount > 0 {
		kv("浮动奖金注数", Yellow(fmt.Sprintf("%d 注（待官方核定）", s.FloatingCount)))
	} else {
		kv("浮动奖金注数", "0 注")
	}
	kv("可确定净收益", fmt.Sprintf("%d 元（固定 %d − 成本 %d）", s.NetDeterminable(), s.FixedTotal, s.TotalCost))
	if s.NetFinalKnown() {
		net := s.NetDeterminable()
		if net >= 0 {
			kv("最终净收益", Green(fmt.Sprintf("+%d 元（盈利）", net)))
		} else {
			kv("最终净收益", Yellow(fmt.Sprintf("%d 元（亏损）", net)))
		}
	} else {
		kv("最终净收益", Yellow("无法准确计算：含浮动奖金注，待官方核定"))
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, Dim("  特别规定（福运奖）说明：本模拟器未执行特别规定。官方规则为"))
	fmt.Fprintln(out, Dim("  奖池资金 ≥15 亿元时生效、某期开奖后跌破 3 亿元时停止；本地程序"))
	fmt.Fprintln(out, Dim("  无法获知真实奖池，故固定不启用，也不提供开关。"))
}
