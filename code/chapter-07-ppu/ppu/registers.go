package ppu

// MARK: 定数定義
const (
	CONTROL_REG_NAMETABLE1_POS uint8 = iota
	CONTROL_REG_NAMETABLE2_POS
	CONTROL_REG_VRAM_ADDR_INC_POS
	CONTROL_REG_SP_PATTERN_ADDR_POS
	CONTROL_REG_BG_PATTERN_ADDR_POS
	CONTROL_REG_SP_SIZE_POS
	CONTROL_REG_MASTER_SLAVE_POS
	CONTROL_REG_GENERATE_NMI_POS
)

const (
	MASK_REG_GRAYSCALE uint8 = iota
	MASK_REG_LEFTMOST_BG_ENABLE_POS
	MASK_REG_LEFTMOST_SP_ENABLE_POS
	MASK_REG_BG_ENABLE_POS
	MASK_REG_SP_ENABLE_POS
	MASK_REG_EMPHASIZE_RED_POS
	MASK_REG_EMPHASIZE_GREEN_POS
	MASK_REG_EMPHASIZE_BLUE_POS
)

const (
	STATUS_REG_SPRITE_OVERFLOW uint8 = 5
	STATUS_REG_SPRITE_ZERO_HIT uint8 = 6
	STATUS_REG_VBLANK_FLAG     uint8 = 7
)

// MARK: コントロールレジスタ ($2000)
type ControlRegister struct {
	/*
		PPU コントロールレジスタ

		7 6 5 4 3 2 1 0 ビット
		------- -------

		V P H B S I N N
		| | | | | | L |
		| | | | | |   +- N: ネームテーブルの基準アドレス
		| | | | | |         (0 = $2000; 1 = $2400; 2 = $2800; 3 = $2C00)
		| | | | | +----- I: VRAMアドレスの増分 (CPU の PPUDATA 読み書き毎)
		| | | | |           (0: +1, VRAM上での横方向; 1: +32, VRAM上での縦方向)
		| | | | +------- S: スプライトのパターンテーブルアドレス (8x8のスプライトのみ)
		| | | |             (0: $0000; 1: $1000; 8x16モードでは不使用)
		| | | +--------- B: 背景のパターンテーブルアドレス (0: $0000; 1: $1000)
		| | +----------- H: スプライトサイズ (0: 8x8 px; 1: 8x16 px)
		| +------------- P: PPU マスター/スレーブの選択
		+--------------- V: Vblank開始時に NMI を発生させるか否か (0: off, 1: on)
	*/

	nameTable1               bool
	nameTable2               bool
	vramAddressIncrement     bool
	spritePatternAddress     bool
	backgroundPatternAddress bool
	spriteSize               bool
	masterSlaveSelect        bool
	generateNMI              bool
}

// MARK: コントロールレジスタのコンストラクタ
func NewControlRegister() ControlRegister {
	return ControlRegister{
		nameTable1:               false,
		nameTable2:               false,
		vramAddressIncrement:     false,
		spritePatternAddress:     false,
		backgroundPatternAddress: false,
		spriteSize:               false,
		masterSlaveSelect:        false,
		generateNMI:              false,
	}
}

// MARK: マスクレジスタ ($2001)
type MaskRegister struct {
	/*
		PPU マスクレジスタ
		7 6 5 4 3 2 1 0
		------- -------
		B G R s b M m G
		| | | | | | | |
		| | | | | | | +- G: カラー/モノクロフラグ (0: カラー, 1: モノクロ)
		| | | | | | +--- m: 画面左端8pxの背景描画 (0: 非表示, 1: 表示)
		| | | | | +----- M: 画面左端8pxのスプライト描画 (0: 非表示, 1: 表示)
		| | | | +------- b: 背景の描画 (0: 非表示, 1: 表示)
		| | | +--------- s: スプライトの描画 (0: 非表示, 1: 表示)
		| | +----------- R: 赤色を強調 (0: 強調しない, 1: 強調する)
		| +------------- G: 緑色を強調 (0: 強調しない, 1: 強調する)
		+--------------- B: 青色を強調 (0: 強調しない, 1: 強調する)
	*/

	grayscale                bool
	leftmostBackgroundEnable bool
	leftmostSpriteEnable     bool
	backgroundEnable         bool
	spriteEnable             bool
	emphasizeRed             bool
	emphasizeGreen           bool
	emphasizeBlue            bool
}

// MARK: マスクレジスタのコンストラクタ
func NewMaskRegister() MaskRegister {
	return MaskRegister{
		grayscale:                false,
		leftmostBackgroundEnable: false,
		leftmostSpriteEnable:     false,
		backgroundEnable:         false,
		spriteEnable:             false,
		emphasizeRed:             false,
		emphasizeGreen:           false,
		emphasizeBlue:            false,
	}
}

// MARK: ステータスレジスタ ($2002)
type StatusRegister struct {
	/*
		PPU ステータスレジスタ

		7 6 5 4 3 2 1 0
		------- -------
		V S O - - - - -
		| | |
		| | |
		| | +------- O: スプライトのオーバーフローフラグ
		| +--------- S: スプライト 0 ヒットフラグ
		+----------- V: VBlank フラグ
	*/

	spriteOverflow bool
	spriteZeroHit  bool
	vBlankFlag     bool
}

// MARK: ステータスレジスタのコンストラクタ
func NewStatusRegister() StatusRegister {
	return StatusRegister{
		spriteOverflow: false,
		spriteZeroHit:  false,
		vBlankFlag:     false,
	}
}

// MARK: アドレスレジスタ [V/Tレジスタ] (PPU内部)
type AddressRegiseter struct {
	/*
		PPU アドレスレジスタ

		16            8  7              0
		---------------  ----------------
		y y y  N N  Y Y  Y Y Y  X X X X X
		L + |  L |  L +  + + |  L + + + |
		    |    |           |          +- X: タイルの画面内列番号 X (0-31)
		    |    |           +------------ Y: タイルの画面内行番号 Y (0-29)
		    |    +------------------------ N: ネームテーブル選択
		    +----------------------------- y: タイル内のY座標 (0-7)
	*/

	coarseX   uint8
	coarseY   uint8
	nameTable uint8
	fineY     uint8
}

// MARK: アドレスレジスタのコンストラクタ
func NewAddressRegister() AddressRegiseter {
	return AddressRegiseter{
		coarseX:   0x00,
		coarseY:   0x00,
		nameTable: 0x00,
		fineY:     0x00,
	}
}

// MARK: Xレジスタ (PPU内部)
type XRegister struct {
	/*
		PPU Xレジスタ

		7 6 5 4 3 2 1 0
		------- -------
		- - - - - X X X
		          L + |
		              +- X: スクロールX座標
	*/

	fineX uint8
}

// MARK: Xレジスタのコンストラクタ
func NewXRegister() XRegister {
	return XRegister{
		fineX: 0x00,
	}
}

// MARK: Wレジスタ (PPU内部)
type WRegister struct {
	/*
		PPU Wレジスタ

		7 6 5 4 3 2 1 0
		------- -------
		- - - - - - - W
		              |
		              +- W: 書き込みラッチ
	*/

	latch bool
}

// MARK: Wレジスタのコンストラクタ
func NewWRegister() WRegister {
	return WRegister{
		latch: false,
	}
}
