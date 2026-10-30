// Package randx 提供基于 Linux 内核随机设备的随机数读取模块。
//
// 设计要点：
//   - 随机数据仅来源于 /dev/random（Linux 内核 CSPRNG），全程不使用 math/rand；
//   - 读取循环正确处理短读（short read）与 EINTR；
//   - 单次读取设置整体超时：/dev/random 长时间阻塞时自动回退到
//     /dev/urandom（同为内核 CSPRNG，仅阻塞语义不同），并通过回调如实提示；
//   - Intn 采用拒绝采样（rejection sampling）消除直接取模的分布偏差；
//   - 通过 Entropy 接口抽象底层熵源，便于单元测试注入确定性数据。
package randx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

const (
	// DefaultTimeout 为单次完整读取的默认超时时间。
	DefaultTimeout = 10 * time.Second

	// RandomDevice 为主随机设备。
	RandomDevice = "/dev/random"

	// UrandomDevice 为阻塞超时后的备用随机设备。
	UrandomDevice = "/dev/urandom"
)

// ErrBlocked 表示随机设备在超时时间内未能返回足够数据（读取阻塞）。
var ErrBlocked = errors.New("随机设备读取超时（阻塞）")

// ErrClosed 表示随机源已关闭。
var ErrClosed = errors.New("随机源已关闭")

// Entropy 为底层熵源接口，*os.File 天然满足该接口；
// 测试中可注入桩实现以构造确定性场景。
type Entropy interface {
	Read(p []byte) (int, error)
}

// Source 是并发安全的随机源，整个程序生命周期使用同一实例。
type Source struct {
	mu sync.Mutex // 序列化读取与设备切换

	primary     Entropy // 主设备（/dev/random）
	primaryPath string

	fallback     Entropy // 备用设备（/dev/urandom）；仅测试构造时可为 nil
	fallbackPath string  // 为空表示不允许回退

	dev     Entropy // 当前实际使用的熵源
	devPath string

	timeout time.Duration

	fellBack atomic.Bool
	closed   atomic.Bool

	closeOnce sync.Once
	closeErr  error

	// OnFallback 在因阻塞回退到备用设备时被回调（用于界面提示），可为 nil。
	OnFallback func(from, to string, waited time.Duration)
}

// New 创建生产环境随机源：/dev/random 为主，阻塞超时后回退 /dev/urandom。
// 两个设备在启动时一次性打开，避免运行期打开与关闭操作产生数据竞争。
func New() (*Source, error) {
	primary, err := os.Open(RandomDevice)
	if err != nil {
		return nil, fmt.Errorf("打开 %s 失败: %w", RandomDevice, err)
	}
	fallback, err := os.Open(UrandomDevice)
	if err != nil {
		primary.Close()
		return nil, fmt.Errorf("打开备用设备 %s 失败: %w", UrandomDevice, err)
	}
	return &Source{
		primary:      primary,
		primaryPath:  RandomDevice,
		fallback:     fallback,
		fallbackPath: UrandomDevice,
		dev:          primary,
		devPath:      RandomDevice,
		timeout:      DefaultTimeout,
	}, nil
}

// newSource 构造可注入熵源的随机源（仅供本包测试使用）。
// fallback 为 nil 且 fallbackPath 非空时，将在首次回退时尝试按路径打开。
func newSource(primary Entropy, primaryPath string, fallback Entropy, fallbackPath string, timeout time.Duration) *Source {
	return &Source{
		primary:      primary,
		primaryPath:  primaryPath,
		fallback:     fallback,
		fallbackPath: fallbackPath,
		dev:          primary,
		devPath:      primaryPath,
		timeout:      timeout,
	}
}

// ActiveDevice 返回当前实际使用的设备路径。
func (s *Source) ActiveDevice() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.devPath
}

// FellBack 报告是否已因阻塞回退到备用设备。
func (s *Source) FellBack() bool { return s.fellBack.Load() }

// Close 关闭随机源并释放底层文件资源；可安全地重复调用。
// Close 不等待进行中的读取：阻塞中的读取会因文件关闭而立即以错误返回。
func (s *Source) Close() error {
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		var errs []error
		for _, e := range []Entropy{s.primary, s.fallback} {
			if e == nil {
				continue
			}
			if c, ok := e.(io.Closer); ok {
				if err := c.Close(); err != nil {
					errs = append(errs, err)
				}
			}
		}
		s.closeErr = errors.Join(errs...)
	})
	return s.closeErr
}

// ReadN 从当前随机设备精确读取 n 字节。
// 处理短读与 EINTR；若主设备阻塞超时且配置了备用设备，则回退后重试一次。
func (s *Source) ReadN(n int) ([]byte, error) {
	if n < 0 {
		return nil, fmt.Errorf("randx: n 必须非负，当前 %d", n)
	}
	if n == 0 {
		return []byte{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed.Load() {
		return nil, ErrClosed
	}

	start := time.Now()
	// 每次尝试使用独立缓冲区，避免超时后遗留的读取协程与后续写入产生竞争。
	buf := make([]byte, n)
	err := readFull(s.dev, buf, s.timeout)
	if err == nil {
		return buf, nil
	}
	if !errors.Is(err, ErrBlocked) || s.fallbackPath == "" || s.devPath == s.fallbackPath {
		return nil, fmt.Errorf("读取 %s 失败: %w", s.devPath, err)
	}

	// 主设备阻塞超时：切换到备用设备并重试一次（不做无限重试）。
	if ferr := s.switchFallbackLocked(); ferr != nil {
		return nil, fmt.Errorf("%s 读取阻塞 %.1fs，且启用备用设备 %s 失败: %w",
			s.devPath, time.Since(start).Seconds(), s.fallbackPath, ferr)
	}
	if s.OnFallback != nil {
		s.OnFallback(s.primaryPath, s.fallbackPath, time.Since(start))
	}
	buf = make([]byte, n)
	if err := readFull(s.dev, buf, s.timeout); err != nil {
		return nil, fmt.Errorf("读取备用设备 %s 失败: %w", s.devPath, err)
	}
	return buf, nil
}

// switchFallbackLocked 切换到备用设备（调用方需持有 s.mu）。
func (s *Source) switchFallbackLocked() error {
	if s.fallback == nil {
		f, err := os.Open(s.fallbackPath)
		if err != nil {
			return fmt.Errorf("打开 %s 失败: %w", s.fallbackPath, err)
		}
		s.fallback = f
	}
	s.dev = s.fallback
	s.devPath = s.fallbackPath
	s.fellBack.Store(true)
	return nil
}

// readFull 从 dev 精确读满 buf：处理短读与 EINTR；
// 超过 timeout 仍未读满时返回 ErrBlocked（不中断仍在进行的底层读取，
// 超时后遗留的读取协程可向带缓冲通道发送并正常退出，不会泄漏）。
func readFull(dev Entropy, buf []byte, timeout time.Duration) error {
	type readResult struct{ err error }
	ch := make(chan readResult, 1)
	go func() {
		total := 0
		for total < len(buf) {
			n, err := dev.Read(buf[total:])
			if n > 0 {
				total += n
			}
			if err != nil {
				if errors.Is(err, syscall.EINTR) {
					continue // 被信号中断，重试
				}
				ch <- readResult{err}
				return
			}
		}
		ch <- readResult{}
	}()
	select {
	case r := <-ch:
		return r.err
	case <-time.After(timeout):
		return ErrBlocked
	}
}

// Uint64 返回一个 64 位均匀随机数。
func (s *Source) Uint64() (uint64, error) {
	b, err := s.ReadN(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

// Intn 返回 [0, n) 内的均匀无偏随机整数。
//
// 采用拒绝采样：仅接受落在最大的、能被 n 整除的取值区间内的值，
// 避免 v % n 在 2^64 不能被 n 整除时引入的取模分布偏差。
func (s *Source) Intn(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("randx: n 必须为正数，当前 %d", n)
	}
	if n == 1 {
		return 0, nil // 结果唯一，无需消耗熵
	}
	bound := uint64(n)
	// rem = 2^64 mod n，借助 MaxUint64 换算，避免 2^64 溢出。
	rem := (math.MaxUint64%bound + 1) % bound
	if rem == 0 {
		// 2^64 恰能被 n 整除，任意值取模均无偏。
		v, err := s.Uint64()
		if err != nil {
			return 0, err
		}
		return int(v % bound), nil
	}
	limit := math.MaxUint64 - (rem - 1) // 接受 v ∈ [0, 2^64-rem)
	for {
		v, err := s.Uint64()
		if err != nil {
			return 0, err
		}
		if v < limit {
			return int(v % bound), nil
		}
		// v 落入尾部“余数区间”，丢弃重取以保证无偏。
	}
}

// UniqueInts 从 [1, max] 中无偏抽取 count 个互不相同的整数。
// 采用部分 Fisher-Yates 抽样：每抽走一个号码，用池尾部未用的号码填补，
// 保证每个号码至多被抽中一次，且组合分布均匀。
func (s *Source) UniqueInts(max, count int) ([]int, error) {
	if max < 1 {
		return nil, fmt.Errorf("randx: max 必须 ≥1，当前 %d", max)
	}
	if count < 0 || count > max {
		return nil, fmt.Errorf("randx: count 须在 [0, max] 内，当前 count=%d, max=%d", count, max)
	}
	pool := make([]int, max)
	for i := range pool {
		pool[i] = i + 1
	}
	picks := make([]int, 0, count)
	for i := 0; i < count; i++ {
		j, err := s.Intn(max - i)
		if err != nil {
			return nil, err
		}
		picks = append(picks, pool[j])
		pool[j] = pool[max-1-i]
	}
	return picks, nil
}
