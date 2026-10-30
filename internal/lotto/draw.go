package lotto

import "fmt"

// NewDraw 生成一期模拟开奖号码：6 个不重复红球（1-33）与 1 个蓝球（1-16），
// 与机选投注共用同一随机源（Randomizer），保证随机行为一致、可测试。
//
// 开奖号码在返回前已一次性生成完毕；随后的终端动画仅负责呈现，
// 不会重新取数，因此不影响随机数据的正确性。
func NewDraw(r Randomizer) (DrawResult, error) {
	t, err := AutoPick(r)
	if err != nil {
		return DrawResult{}, fmt.Errorf("生成开奖号码失败: %w", err)
	}
	return DrawResult{Reds: t.Reds, Blue: t.Blue}, nil
}
