// 双色球本地模拟器（SSQ Simulator）。
//
// 本程序是纯本地随机模拟器：
//   - 不连接福利彩票官方系统；
//   - 不读取、不伪造真实开奖号码；
//   - 不宣称能够预测真实开奖；
//   - 浮动奖金（一、二等奖）一律显示“待官方核定”，不编造金额；
//   - 福运奖依据官方特别规定条件（奖池 ≥15 亿元生效、跌破 3 亿元停止），
//     本地无法获知奖池，固定按“未执行特别规定”处理，不提供开关。
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ssq/internal/lotto"
	"ssq/internal/randx"
	"ssq/internal/tui"
)

// exitInterrupt 为收到 SIGINT/SIGTERM 时的退出码（128+2，遵循 shell 惯例）。
const exitInterrupt = 130

// 编译期断言：randx.Source 实现 lotto.Randomizer。
var _ lotto.Randomizer = (*randx.Source)(nil)

func main() {
	os.Exit(run())
}

// run 组织主流程并返回进程退出码。
func run() int {
	// 手动接管 SIGINT/SIGTERM：真实信号到来时才执行退出流程。
	// 不使用 signal.NotifyContext 的 stop()，因为它同样会取消 context，
	// 导致程序正常结束时误触发中断退出协程。
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	// 初始化随机源：/dev/random 为主，阻塞超时回退 /dev/urandom。
	src, err := randx.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, tui.Red("初始化随机源失败: "+err.Error()))
		return 1
	}
	defer src.Close()

	// Ctrl+C 优雅退出：取消长操作（动画等）、提示、关闭随机源、以 130 退出。
	go func() {
		<-sigCh
		cancel()
		_ = src.Close()
		fmt.Fprintln(os.Stdout)
		fmt.Fprintln(os.Stdout, tui.Yellow("收到中断信号（Ctrl+C），已安全退出。"))
		os.Exit(exitInterrupt)
	}()

	// 随机源阻塞回退时向用户如实提示。
	src.OnFallback = func(from, to string, waited time.Duration) {
		tui.Warn(fmt.Sprintf("%s 读取阻塞 %.1f 秒，已回退到 %s（同为内核 CSPRNG）",
			from, waited.Seconds(), to))
	}

	tui.Welcome(src.ActiveDevice())
	in := bufio.NewReader(os.Stdin)

	// 阶段一：选号（手动/机选，最多 5 注）并确认。
	tickets, err := selectionPhase(in, src)
	if err != nil {
		switch {
		case errors.Is(err, tui.ErrQuit):
			tui.Info("已退出，感谢使用。")
		case errors.Is(err, tui.ErrEOF):
			tui.Info("检测到输入结束（EOF），已安全退出。")
		default:
			tui.Warn("发生错误: " + err.Error())
			return 1
		}
		return 0
	}

	// 阶段二：生成开奖号码（先一次性取数，再做揭晓动画）。
	draw, err := lotto.NewDraw(src)
	if err != nil {
		tui.Warn("生成开奖号码失败: " + err.Error())
		return 1
	}
	if err := tui.RevealDraw(ctx, draw); err != nil {
		if ctx.Err() != nil {
			return exitInterrupt // 通常已由中断监听协程完成退出
		}
		tui.Warn("开奖动画异常: " + err.Error())
		return 1
	}

	// 阶段三：对奖与统计报告。
	result := lotto.JudgeAll(tickets, draw, false)
	stats := lotto.CalcStats(result)
	tui.Report(result, stats)
	tui.Info("本次模拟结束。理性购彩，量力而行。")
	return 0
}

// selectionPhase 为选号阶段：菜单循环，录入或机选最多 lotto.MaxTickets 注，
// 直到用户确认投注、退出或输入结束。
func selectionPhase(in *bufio.Reader, src *randx.Source) ([]lotto.Ticket, error) {
	tickets := make([]lotto.Ticket, 0, lotto.MaxTickets)
	for {
		tui.Menu(len(tickets))
		choice, err := tui.Prompt(in, tui.Cyan("请选择操作: "))
		if err != nil {
			return nil, err
		}
		switch strings.TrimSpace(choice) {
		case "1": // 手动选号
			if len(tickets) >= lotto.MaxTickets {
				tui.Warn(fmt.Sprintf("已达到单次模拟上限 %d 注。", lotto.MaxTickets))
				continue
			}
			t, err := askTicket(in)
			if err != nil {
				return nil, err
			}
			tickets = append(tickets, t)
			tui.ShowTickets(tickets)
		case "2", "3": // 机选一注 / 机选补满
			want := 1
			if choice == "3" {
				want = lotto.MaxTickets - len(tickets)
			}
			if len(tickets) >= lotto.MaxTickets || want <= 0 {
				tui.Warn(fmt.Sprintf("已达到单次模拟上限 %d 注。", lotto.MaxTickets))
				continue
			}
			for i := 0; i < want; i++ {
				t, err := lotto.AutoPick(src)
				if err != nil {
					return nil, fmt.Errorf("机选失败: %w", err)
				}
				tickets = append(tickets, t)
			}
			tui.Info(fmt.Sprintf("已机选 %d 注。", want))
			tui.ShowTickets(tickets)
		case "4": // 清空重选
			if len(tickets) == 0 {
				tui.Warn("当前没有已选号码。")
				continue
			}
			tickets = tickets[:0]
			tui.Info("已清空全部号码，请重新选号。")
		case "5": // 确认投注
			if len(tickets) == 0 {
				tui.Warn("请先通过选项 1/2/3 选择至少 1 注号码。")
				continue
			}
			confirmed, err := confirmPhase(in, &tickets)
			if err != nil {
				return nil, err
			}
			if confirmed {
				return tickets, nil
			}
		case "0", "q", "Q":
			return nil, tui.ErrQuit
		default:
			tui.Warn("无效选项 “" + strings.TrimSpace(choice) + "”，请输入菜单中的编号。")
		}
	}
}

// confirmPhase 展示全部投注号码并请求确认：
// y=确认开奖；r=重新选号（清空后返回菜单）；q=退出。
func confirmPhase(in *bufio.Reader, tickets *[]lotto.Ticket) (bool, error) {
	for {
		tui.ConfirmView(*tickets)
		ans, err := tui.Prompt(in, tui.Cyan("确认开奖 [y] / 重新选号 [r] / 退出 [q]: "))
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(ans)) {
		case "y", "yes":
			return true, nil
		case "r":
			*tickets = (*tickets)[:0]
			tui.Info("已清空号码，返回选号菜单。")
			return false, nil
		case "q":
			return false, tui.ErrQuit
		default:
			tui.Warn("请输入 y、r 或 q。")
		}
	}
}

// askTicket 交互式录入一注号码：先红球后蓝球；
// 严格校验数量、范围、重复与格式，错误时提示原因并允许重新输入。
func askTicket(in *bufio.Reader) (lotto.Ticket, error) {
	redLabel := tui.Yellow(fmt.Sprintf("红球（%02d-%02d 共 6 个，空格或逗号分隔）: ",
		lotto.RedMin, lotto.RedMax))
	var reds []int
	for reds == nil {
		line, err := tui.Prompt(in, redLabel)
		if err != nil {
			return lotto.Ticket{}, err
		}
		rs, perr := lotto.ParseReds(line)
		if perr != nil {
			tui.Warn("红球输入有误: " + perr.Error())
			continue
		}
		reds = rs
	}

	blueLabel := tui.Yellow(fmt.Sprintf("蓝球（%02d-%02d 共 1 个）: ",
		lotto.BlueMin, lotto.BlueMax))
	var blue int
	for blue == 0 {
		line, err := tui.Prompt(in, blueLabel)
		if err != nil {
			return lotto.Ticket{}, err
		}
		b, perr := lotto.ParseBlue(line)
		if perr != nil {
			tui.Warn("蓝球输入有误: " + perr.Error())
			continue
		}
		blue = b
	}
	return lotto.NewTicket(reds, blue)
}
