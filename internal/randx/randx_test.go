package randx

import (
	"bytes"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"
	"time"
)

// seqEntropy 顺序输出预置字节；short 为 true 时每次最多输出 3 字节以模拟短读；
// err 非空时在数据耗尽后返回该错误。
type seqEntropy struct {
	data  []byte
	pos   int
	short bool
	err   error
}

func (e *seqEntropy) Read(p []byte) (int, error) {
	if e.err != nil && e.pos >= len(e.data) {
		return 0, e.err
	}
	avail := len(e.data) - e.pos
	if avail <= 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > avail {
		n = avail
	}
	if e.short && n > 3 {
		n = 3
	}
	copy(p, e.data[e.pos:e.pos+n])
	e.pos += n
	return n, nil
}

// counterEntropy 输出简单确定性递增字节序列（仅测试用，不涉及随机源规则）。
type counterEntropy struct{ next byte }

func (e *counterEntropy) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = e.next
		e.next += 7
	}
	return len(p), nil
}

// intrEntropy 前 intr 次 Read 返回 EINTR，随后按短读方式输出数据。
type intrEntropy struct {
	seqEntropy
	intr  int
	calls int
}

func (e *intrEntropy) Read(p []byte) (int, error) {
	e.calls++
	if e.calls <= e.intr {
		return 0, syscall.EINTR
	}
	return e.seqEntropy.Read(p)
}

// blockingEntropy 永久阻塞，直到 release 被调用。
type blockingEntropy struct{ gate chan struct{} }

func newBlockingEntropy() *blockingEntropy {
	return &blockingEntropy{gate: make(chan struct{})}
}

func (e *blockingEntropy) Read(p []byte) (int, error) {
	<-e.gate
	return 0, io.EOF
}

func (e *blockingEntropy) release() { close(e.gate) }

func TestReadNHandlesShortReadsAndEINTR(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	e := &intrEntropy{
		seqEntropy: seqEntropy{data: data, short: true},
		intr:       2,
	}
	src := newSource(e, "fake", nil, "", time.Second)
	got, err := src.ReadN(8)
	if err != nil {
		t.Fatalf("ReadN: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("读取数据不一致: %v", got)
	}
}

func TestReadNPropagatesError(t *testing.T) {
	src := newSource(&seqEntropy{err: syscall.EIO}, "fake", nil, "", time.Second)
	if _, err := src.ReadN(8); err == nil {
		t.Fatal("底层读取错误应当向上传播")
	}
}

func TestIntnRejectionSampling(t *testing.T) {
	// n=3 时 2^64 mod 3 == 1，仅 MaxUint64 落入拒绝区间；
	// 第一次取 MaxUint64（被拒绝），第二次取 7 → 7%3 == 1。
	data := append(bytes.Repeat([]byte{0xFF}, 8), 0x07, 0, 0, 0, 0, 0, 0, 0)
	src := newSource(&seqEntropy{data: data}, "fake", nil, "", time.Second)
	got, err := src.Intn(3)
	if err != nil {
		t.Fatalf("Intn: %v", err)
	}
	if got != 1 {
		t.Fatalf("拒绝采样结果期望 1，实际 %d", got)
	}
}

func TestIntnAllAcceptedWhenDivisible(t *testing.T) {
	// n=2 时 2^64 mod 2 == 0，所有值均可直接接受：7%2 == 1。
	data := []byte{7, 0, 0, 0, 0, 0, 0, 0}
	src := newSource(&seqEntropy{data: data}, "fake", nil, "", time.Second)
	got, err := src.Intn(2)
	if err != nil || got != 1 {
		t.Fatalf("期望 1，实际 %d（err=%v）", got, err)
	}
}

func TestIntnErrorsWhenEntropyExhausted(t *testing.T) {
	// 仅有 MaxUint64 可读（n=3 时必然被拒绝），熵耗尽后应返回错误。
	src := newSource(&seqEntropy{data: bytes.Repeat([]byte{0xFF}, 8)}, "fake", nil, "", time.Second)
	if _, err := src.Intn(3); err == nil {
		t.Fatal("熵耗尽时应返回错误而非死循环")
	}
}

func TestIntnStaysInRange(t *testing.T) {
	src := newSource(&counterEntropy{}, "fake", nil, "", time.Second)
	for i := 0; i < 200; i++ {
		v, err := src.Intn(33)
		if err != nil {
			t.Fatalf("第 %d 次调用: %v", i, err)
		}
		if v < 0 || v >= 33 {
			t.Fatalf("结果越界: %d", v)
		}
	}
}

func TestUniqueIntsDistinctAndInRange(t *testing.T) {
	for i := 0; i < 50; i++ {
		src := newSource(&counterEntropy{next: byte(i * 5)}, "fake", nil, "", time.Second)
		got, err := src.UniqueInts(33, 6)
		if err != nil {
			t.Fatalf("第 %d 组: %v", i, err)
		}
		if len(got) != 6 {
			t.Fatalf("第 %d 组数量错误: %v", i, got)
		}
		seen := make(map[int]bool, 6)
		for _, v := range got {
			if v < 1 || v > 33 {
				t.Fatalf("第 %d 组越界号码: %d", i, v)
			}
			if seen[v] {
				t.Fatalf("第 %d 组重复号码: %d", i, v)
			}
			seen[v] = true
		}
	}
}

func TestInvalidParams(t *testing.T) {
	src := newSource(&counterEntropy{}, "fake", nil, "", time.Second)
	if _, err := src.UniqueInts(5, 6); err == nil {
		t.Fatal("count > max 应报错")
	}
	if _, err := src.UniqueInts(0, 1); err == nil {
		t.Fatal("max < 1 应报错")
	}
	if _, err := src.Intn(0); err == nil {
		t.Fatal("n = 0 应报错")
	}
	if v, err := src.Intn(1); err != nil || v != 0 {
		t.Fatalf("n = 1 应恒返回 0，实际 (%d, %v)", v, err)
	}
}

func TestBlockedFallbackToUrandom(t *testing.T) {
	blk := newBlockingEntropy()
	defer blk.release()
	good := &counterEntropy{}

	var fbFrom, fbTo string
	var fbWaited time.Duration
	src := newSource(blk, "/dev/random", good, "/dev/urandom", 40*time.Millisecond)
	src.OnFallback = func(from, to string, waited time.Duration) {
		fbFrom, fbTo, fbWaited = from, to, waited
	}

	got, err := src.ReadN(8)
	if err != nil {
		t.Fatalf("回退后 ReadN: %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("回退后数据长度错误: %d", len(got))
	}
	if !src.FellBack() {
		t.Fatal("应标记已回退")
	}
	if dev := src.ActiveDevice(); dev != "/dev/urandom" {
		t.Fatalf("应使用备用设备，实际 %s", dev)
	}
	if fbFrom != "/dev/random" || fbTo != "/dev/urandom" {
		t.Fatalf("回退回调参数错误: %s → %s", fbFrom, fbTo)
	}
	if fbWaited < 40*time.Millisecond {
		t.Fatalf("回退等待时长异常: %v", fbWaited)
	}
}

func TestFallbackOpenFailure(t *testing.T) {
	blk := newBlockingEntropy()
	defer blk.release()
	src := newSource(blk, "/dev/random", nil, "/definitely/not/exist", 30*time.Millisecond)
	if _, err := src.ReadN(4); err == nil {
		t.Fatal("备用设备不可用且主设备阻塞时应当报错")
	}
}

func TestCloseIdempotentAndRejectsAfterClose(t *testing.T) {
	src := newSource(&counterEntropy{}, "fake", nil, "", time.Second)
	if err := src.Close(); err != nil {
		t.Fatalf("第一次 Close: %v", err)
	}
	if err := src.Close(); err != nil {
		t.Fatalf("第二次 Close: %v", err)
	}
	if _, err := src.ReadN(4); !errors.Is(err, ErrClosed) {
		t.Fatalf("关闭后读取应返回 ErrClosed，实际 %v", err)
	}
}

func TestLiveURandom(t *testing.T) {
	f, err := os.Open(UrandomDevice)
	if err != nil {
		t.Skipf("跳过：无法打开 %s: %v", UrandomDevice, err)
	}
	defer f.Close()
	src := newSource(f, UrandomDevice, nil, "", 2*time.Second)

	for i := 0; i < 100; i++ {
		got, err := src.UniqueInts(33, 6)
		if err != nil {
			t.Fatalf("第 %d 组: %v", i, err)
		}
		seen := make(map[int]bool, 6)
		for _, v := range got {
			if v < 1 || v > 33 {
				t.Fatalf("第 %d 组越界: %d", i, v)
			}
			if seen[v] {
				t.Fatalf("第 %d 组重复: %d", i, v)
			}
			seen[v] = true
		}
	}
	for i := 0; i < 100; i++ {
		v, err := src.Intn(16)
		if err != nil {
			t.Fatalf("Intn: %v", err)
		}
		if v < 0 || v >= 16 {
			t.Fatalf("Intn 越界: %d", v)
		}
	}
}
