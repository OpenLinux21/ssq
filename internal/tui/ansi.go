// Package tui 负责终端界面渲染与交互：ANSI 颜色、对齐排版、
// 逐球开奖动画与输入提示。
//
// 仅使用标准 ANSI 转义序列与标准库，不依赖任何终端 UI 框架；
// bash、Fish Shell 等现代 Linux 终端均可正常显示。
// 设置 NO_COLOR 环境变量或标准输出不是终端时，自动禁用颜色。
package tui

import (
	"fmt"
	"os"
	"strings"
)

// ansiReset 重置所有属性。
const ansiReset = "\x1b[0m"

// colorEnabled 为颜色开关，包初始化时根据环境判定。
var colorEnabled = detectColor()

func detectColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// 仅在标准输出为字符设备（终端）时启用颜色，保证管道/重定向输出干净。
	return fi.Mode()&os.ModeCharDevice != 0
}

// wrap 用指定 ANSI 属性包裹字符串。
func wrap(code, s string) string {
	if !colorEnabled || s == "" {
		return s
	}
	return "\x1b[" + code + "m" + s + ansiReset
}

// 语义化颜色函数：红球红色、蓝球蓝色、中奖信息绿色、提示黄/青色。
func Red(s string) string    { return wrap("91", s) }
func Blue(s string) string   { return wrap("94", s) }
func Green(s string) string  { return wrap("92", s) }
func Yellow(s string) string { return wrap("93", s) }
func Cyan(s string) string   { return wrap("96", s) }
func Bold(s string) string   { return wrap("1", s) }
func Dim(s string) string    { return wrap("2", s) }

// RedBall 渲染红球号码芯片（亮白字 + 亮红底，两位数字）。
func RedBall(n int) string { return ball(n, "97;101") }

// BlueBall 渲染蓝球号码芯片（亮白字 + 亮蓝底，两位数字）。
func BlueBall(n int) string { return ball(n, "97;104") }

// ball 生成 " 01 " 样式的号码芯片；禁用颜色时退化为纯文本，宽度一致。
func ball(n int, code string) string {
	s := fmt.Sprintf("%02d", n)
	if !colorEnabled {
		return " " + s + " "
	}
	return "\x1b[" + code + "m " + s + " \x1b[0m"
}

// isWide 判断 rune 是否为双列显示字符（CJK 及全角符号等的近似判定）。
func isWide(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // 谚文字母
		r >= 0x2E80 && r <= 0x303E, // CJK 部首与符号
		r >= 0x3041 && r <= 0x33FF, // 假名、注音、CJK 兼容
		r >= 0x3400 && r <= 0x4DBF, // CJK 扩展 A
		r >= 0x4E00 && r <= 0x9FFF, // CJK 统一表意
		r >= 0xA960 && r <= 0xA97F,
		r >= 0xAC00 && r <= 0xD7A3, // 谚文音节
		r >= 0xF900 && r <= 0xFAFF, // CJK 兼容表意
		r >= 0xFE30 && r <= 0xFE4F,
		r >= 0xFF00 && r <= 0xFF60, // 全角 ASCII
		r >= 0xFFE0 && r <= 0xFFE6:
		return true
	default:
		return false
	}
}

// displayWidth 计算字符串的近似终端显示宽度：
// 自动忽略 ANSI 转义序列，CJK 等宽字符按 2 列计。
func displayWidth(s string) int {
	w := 0
	inEsc := false
	for _, r := range s {
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		if r == 0x1b {
			inEsc = true
			continue
		}
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// padRight 在右侧补空格至指定显示宽度。
func padRight(s string, w int) string {
	if gap := w - displayWidth(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

// padLeft 在左侧补空格至指定显示宽度。
func padLeft(s string, w int) string {
	if gap := w - displayWidth(s); gap > 0 {
		return strings.Repeat(" ", gap) + s
	}
	return s
}
