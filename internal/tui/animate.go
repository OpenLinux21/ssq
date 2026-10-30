package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ssq/internal/lotto"
)

// revealStep 为逐球揭晓的时间间隔。
const revealStep = 450 * time.Millisecond

// RevealDraw 以逐步揭晓动画展示开奖结果。
//
// 开奖号码在调用前已由随机源一次性生成，动画只负责呈现，
// 不参与随机数据的产生，因此不影响随机数据的正确性。
// 动画可随时通过 ctx（Ctrl+C/SIGTERM）取消。
func RevealDraw(ctx context.Context, d lotto.DrawResult) error {
	fmt.Fprintln(out)
	fmt.Fprint(out, Cyan("摇奖机转动中"))
	for i := 0; i < 4; i++ {
		if err := sleepCtx(ctx, 280*time.Millisecond); err != nil {
			fmt.Fprintln(out)
			return err
		}
		fmt.Fprint(out, Yellow("."))
	}
	fmt.Fprintln(out)

	// 共 7 步：第 1-6 步逐个揭晓红球，第 7 步揭晓蓝球。
	for step := 1; step <= lotto.RedCount+1; step++ {
		paintReveal(d, step, step == lotto.RedCount+1)
		if err := sleepCtx(ctx, revealStep); err != nil {
			fmt.Fprintln(out)
			return err
		}
	}
	return nil
}

// paintReveal 原地重绘一行揭晓画面：已揭晓的号码用芯片渲染，
// 未揭晓位置用同宽占位符；done 为 true 时输出换行定格。
// step 表示已揭晓号码个数（1-6 为红球，7 表示蓝球也已揭晓）。
func paintReveal(d lotto.DrawResult, step int, done bool) {
	var b strings.Builder
	b.WriteString(Bold("开奖号码  "))

	redShown := step
	if redShown > lotto.RedCount {
		redShown = lotto.RedCount
	}
	for i := 0; i < redShown; i++ {
		b.WriteString(RedBall(d.Reds[i]))
		b.WriteString(" ")
	}
	for i := 0; i < lotto.RedCount-redShown; i++ {
		b.WriteString(mask())
		b.WriteString(" ")
	}
	b.WriteString(Dim("┃") + " ")
	if step > lotto.RedCount {
		b.WriteString(BlueBall(d.Blue))
	} else {
		b.WriteString(mask())
	}

	// \r 回行首并清除整行后重绘，实现原地刷新效果。
	fmt.Fprint(out, "\r\x1b[2K"+b.String())
	if done {
		fmt.Fprintln(out)
	}
}

// mask 生成与号码芯片同宽的未揭晓占位符。
func mask() string { return Dim(" ?? ") }

// sleepCtx 为可被 ctx 取消的休眠。
func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
