package apu

import "sync"

// MARK: 定数定義
const (
	BUFFER_SIZE = 2048
)

// MARK: RingBufferの定義
type RingBuffer struct {
	buffer   [BUFFER_SIZE]float32
	writePos int
	readPos  int
	size     int
	mutex    sync.Mutex
}

// MARK: RingBufferのコンストラクタ
func NewRingBuffer() *RingBuffer {
	return &RingBuffer{
		buffer:   [BUFFER_SIZE]float32{},
		writePos: 0,
		readPos:  0,
		mutex:    sync.Mutex{},
	}
}

// MARK: リングバッファの書き込み
func (b *RingBuffer) Write(value float32) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if b.size == BUFFER_SIZE {
		b.readPos = (b.readPos + 1) % BUFFER_SIZE
		b.size--
	}

	b.buffer[b.writePos] = value
	b.writePos = (b.writePos + 1) % BUFFER_SIZE
	b.size++
}

// MARK: リングバッファの読み取り
func (b *RingBuffer) Read(dst []float32) int {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if len(dst) == 0 {
		return 0
	}

	// データがない場合は全て0.0
	if b.size == 0 {
		for i := range dst {
			dst[i] = 0.0
		}
		return 0
	}

	// 読み取るデータ量 (要求された量と利用可能量の小さい方)
	n := min(len(dst), b.size)

	contiguous := BUFFER_SIZE - b.readPos
	if n <= contiguous {
		// 連続領域に一括コピー
		copy(dst, b.buffer[b.readPos:b.readPos+n])
	} else {
		// 連続領域が足りなければ分割コピー
		copy(dst[:contiguous], b.buffer[b.readPos:])
		copy(dst[contiguous:n], b.buffer[:n-contiguous])
	}

	b.readPos = (b.readPos + n) % BUFFER_SIZE
	b.size -= n

	// 不足分は無音で埋める
	for i := n; i < len(dst); i++ {
		dst[i] = 0.0
	}

	return n
}

// MARK: リングバッファの有効領域サイズを取得
func (b *RingBuffer) Available() int {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.size
}
