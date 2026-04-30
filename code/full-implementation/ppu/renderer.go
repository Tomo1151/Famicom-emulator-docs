package ppu

// MARK: 定数定義
const (
	SCREEN_WIDTH  uint = 256
	SCREEN_HEIGHT uint = 240

	SCALE_FACTOR uint   = 3
	FPS          uint64 = 60
)

// MARK: 色型の定義
type rgb [3]uint8

// MARK: Canvasの定義
type Canvas struct {
	width  uint
	height uint
	buffer [SCREEN_WIDTH * SCREEN_HEIGHT * 3]uint8
}

// MARK: Canvasのコンストラクタ
func NewCanvas() Canvas {
	return Canvas{
		width:  SCREEN_WIDTH,
		height: SCREEN_HEIGHT,
	}
}

// MARK: 指定した座標へ色を書き込み
func (c *Canvas) SetPixel(x, y uint, color rgb) {
	basePos := (y*SCREEN_WIDTH + x) * 3 // 基準インデックスの計算
	c.buffer[basePos+0] = color[0]      // R
	c.buffer[basePos+1] = color[1]      // G
	c.buffer[basePos+2] = color[2]      // B
}

// MARK: バッファの取得メソッド
func (c *Canvas) Buffer() []uint8 {
	return c.buffer[:]
}
