package apu

// MARK: 定数定義
const (
	STATUS_REG_IS_1CH_ACTIVE_POS uint8 = iota
	STATUS_REG_IS_2CH_ACTIVE_POS
	STATUS_REG_IS_3CH_ACTIVE_POS
	STATUS_REG_IS_4CH_ACTIVE_POS
	STATUS_REG_IS_5CH_ACTIVE_POS
	STATUS_REG_FRAME_IRQ_POS uint8 = 6
	STATUS_REG_DMC_IRQ_POS   uint8 = 7
)

const (
	FRAME_COUNTER_IRQ_POS  uint8 = 6
	FRAME_COUNTER_MODE_POS uint8 = 7
)

const (
	NOISE_MODE_SHORT NoiseShiftMode = iota
	NOISE_MODE_LONG
)

// MARK: NoiseShiftModeの定義
type NoiseShiftMode uint8

// MARK: 矩形波レジスタ
type SquareWaveRegister struct {
	// $4000 / $4004 エンベロープ有効時
	envelopePeriod  uint8
	envelopeDisable bool
	envelopeLoop    bool

	// エンベロープ無効時
	volume            uint8
	constantVolume    bool
	lengthCounterHalt bool

	duty uint8

	// $4001 / $4005
	sweepShift     uint8
	sweepDirection uint8
	sweepPeriod    uint8
	sweepEnabled   bool

	// $4002 / $4006
	timerLower uint8

	// $4003 / $4007
	timerUpper        uint8
	lengthCounterLoad uint8
}

// MARK: 矩形波レジスタのコンストラクタ
func NewSquareWaveRegister() SquareWaveRegister {
	return SquareWaveRegister{
		envelopePeriod:    0x00,
		envelopeDisable:   false,
		envelopeLoop:      false,
		volume:            0x00,
		constantVolume:    false,
		lengthCounterHalt: false,
		duty:              0x00,
		sweepShift:        0x00,
		sweepDirection:    0x00,
		sweepPeriod:       0x00,
		sweepEnabled:      false,
		timerLower:        0x00,
		timerUpper:        0x00,
		lengthCounterLoad: 0x00,
	}
}

// MARK: 矩形波レジスタへの書き込み
func (swr *SquareWaveRegister) write(address uint16, value uint8) {
	switch address {
	case 0x04000, 0x4004:
		/*
			$4000 / $4004 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			D D H C V V V V
			L | | | L + + |
				| | |       +- V: ボリューム / エンベロープ周期
				| | |
				| | +--------- C: 固定ボリューム / エンベロープ無効
				| +----------- H: 長さカウンタ停止 / エンベロープループ
				+------------- D: デューティ比
				                  (00: 12.5%; 01: 25%; 10: 50%; 11: 75%)
		*/
		swr.envelopePeriod = (value & 0x0F)
		swr.envelopeDisable = (value & 0x10) != 0
		swr.envelopeLoop = (value & 0x20) != 0

		swr.volume = (value & 0x0F)
		swr.constantVolume = (value & 0x10) != 0
		swr.lengthCounterHalt = (value & 0x20) != 0

		swr.duty = (value & 0xC0) >> 6
	case 0x4001, 0x4005:
		/*
			$4001 / $4005 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			E P P P N S S S
			| L + | | L + |
			|	    | |     +- S: スイープシフトカウント
			|     | |
			|	    | +------- N: スイープ方向反転 (0: 低周波方向; 1: 高周波方向)
			|	    +--------- P: スイープ周期
			|
			+--------------- E: スイープ有効フラグ
		*/
		swr.sweepShift = (value & 0x07)
		swr.sweepDirection = (value & 0x08) >> 3
		swr.sweepPeriod = (value & 0x70) >> 4
		swr.sweepEnabled = (value & 0x80) != 0
	case 0x4002, 0x4006:
		/*
			$4002 / $4006 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			L L L L L L L L
			L + + + + + + |
			 	            +- L: タイマ下位8ビット
		*/
		swr.timerLower = value
	case 0x4003, 0x4007:
		/*
			$4003 / $4007 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			l l l l l H H H
			L + + + | L + |
				      |     +- H: タイマ上位3ビット
			 	      |
				      +------- l: 長さカウンタのロード値
		*/
		swr.timerUpper = (value & 0x07)
		swr.lengthCounterLoad = (value & 0xF8) >> 3
	}
}

// MARK: 三角波レジスタ
type TriangleWaveRegister struct {
	// $4008
	lengthCounterHalt   bool
	control             bool
	linearCounterReload uint8

	// $400A
	timerLower uint8

	// $400B
	timerUpper        uint8
	lengthCounterLoad uint8
}

// MARK: 三角波レジスタのコンストラクタ
func NewTriangleWaveRegister() TriangleWaveRegister {
	return TriangleWaveRegister{
		lengthCounterHalt:   false,
		control:             false,
		linearCounterReload: 0x00,
		timerLower:          0x00,
		timerUpper:          0x00,
		lengthCounterLoad:   0x00,
	}
}

// MARK: 三角波レジスタの書き込み
func (twr *TriangleWaveRegister) write(address uint16, value uint8) {
	switch address {
	case 0x4008:
		/*
			$4008 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			C R R R R R R R
			| L + + + + + |
			|	            +- R: 線形カウンタのリロード値
			|
			+--------------- C: コントロールフラグ / 長さカウンタの停止
		*/
		twr.control = (value & 0x80) != 0
		twr.lengthCounterHalt = (value & 0x80) != 0
		twr.linearCounterReload = (value & 0x7F)
	case 0x400A:
		/*
			$400A 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			L L L L L L L L
			L + + + + + + |
			 	            +- L: タイマ下位8ビット
		*/
		twr.timerLower = value
	case 0x400B:
		/*
			$400B 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			l l l l l H H H
			L + + + | L + |
				      |     +- H: タイマ上位3ビット
			 	      |
				      +------- l: 長さカウンタのロード値
		*/
		twr.timerUpper = (value & 0x07)
		twr.lengthCounterLoad = (value & 0xF8) >> 3
	}
}

// MARK: ノイズレジスタ
type NoiseWaveRegister struct {
	// $400C エンベロープ有効時
	envelopePeriod  uint8
	envelopeDisable bool
	envelopeLoop    bool

	// エンベロープ無効時
	volume            uint8
	constantVolume    bool
	lengthCounterHalt bool

	// $400E
	noiseMode        bool
	timerPeriodIndex uint8

	// $400F
	lengthCounterLoad uint8
}

// MARK: ノイズレジスタのコンストラクタ
func NewNoiseWaveRegister() NoiseWaveRegister {
	return NoiseWaveRegister{
		envelopePeriod:    0x00,
		envelopeDisable:   false,
		envelopeLoop:      false,
		volume:            0x00,
		constantVolume:    false,
		lengthCounterHalt: false,
		noiseMode:         false,
		timerPeriodIndex:  0x00,
		lengthCounterLoad: 0x00,
	}
}

// MARK: ノイズレジスタの書き込み
func (nwr *NoiseWaveRegister) write(address uint16, value uint8) {
	switch address {
	case 0x400C:
		/*
			$400C 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			- - H C V V V V
			    | | L + + |
				  | |       +- V: ボリューム / エンベロープ周期
				  | |
				  | +--------- C: 固定ボリューム / エンベロープ無効
				  +----------- H: 長さカウンタ停止 / エンベロープループ
		*/
		nwr.envelopePeriod = (value & 0x0F)
		nwr.envelopeDisable = (value & 0x10) != 0
		nwr.envelopeLoop = (value & 0x20) != 0

		nwr.volume = (value & 0x0F)
		nwr.constantVolume = (value & 0x10) != 0
		nwr.lengthCounterHalt = (value & 0x20) != 0
	case 0x400E:
		/*
			$400E 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			M - - - P P P P
			|       L + + |
			|	            +- P: ノイズシフトレジスタのタイマ周期
			|
			+--------------- M: ノイズモード (0: 長周期; 1: 短周期)
		*/
		nwr.timerPeriodIndex = (value & 0x0F)
		nwr.noiseMode = (value & 0x80) != 0
	case 0x400F:
		/*
			$400B 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			l l l l l - - -
			L + + + |
			 	      |
				      +------- l: 長さカウンタのロード値
		*/
		nwr.lengthCounterLoad = (value & 0xF8) >> 3
	}
}

// MARK: DMCレジスタ
type DMCRegister struct {
	// $4010
	irqEnabled bool
	loop       bool
	rateIndex  uint8

	// $4011
	directLoad uint8

	// $4012
	sampleAddress uint8

	// $4013
	sampleLength uint8
}

// MARK: DMCレジスタのコンストラクタ
func NewDMCRegister() DMCRegister {
	return DMCRegister{
		irqEnabled:    false,
		loop:          false,
		rateIndex:     0x00,
		directLoad:    0x00,
		sampleAddress: 0x00,
		sampleLength:  0x00,
	}
}

// MARK: DMCレジスタの書き込み
func (dr *DMCRegister) write(address uint16, value uint8) {
	switch address {
	case 0x4010:
		/*
			$4010 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			I L - - R R R R
			| |     L + + |
			| |           +- R: 再生速度選択
			| |
			| +------------- L: ループ有効フラグ
			+--------------- I: 割り込み有効フラグ
		*/
		dr.rateIndex = (value & 0x0F)
		dr.loop = (value & 0x40) != 0
		dr.irqEnabled = (value & 0x80) != 0
	case 0x4011:
		/*
			$4011 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			- D D D D D D D
			  L + + + + + |
			 	            +- D: 直接読み込む値
		*/
		dr.directLoad = (value & 0x7F)
	case 0x4012:
		/*
			$4012 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			A A A A A A A A
			L + + + + + + |
			 	            +- A: サンプルアドレス開始アドレス
		*/
		dr.sampleAddress = value
	case 0x4013:
		/*
			$4013 書き込み

			7 6 5 4 3 2 1 0 ビット
			------- -------

			L L L L L L L L
			L + + + + + + |
			 	            +- L: サンプルの長さ
		*/
		dr.sampleLength = value
	}
}

// MARK: ステータスレジスタの定義
type StatusRegister struct {
	/*
		APU ステータスレジスタ

		7 6 5 4 3 2 1 0
		| | | | | | | |
		| | | | | | | +- 1chが再生中かどうか
		| | | | | | +--- 2chが再生中かどうか
		| | | | | +----- 3chが再生中かどうか
		| | | | +------- 4chが再生中かどうか
		| | | +--------- 5chが再生中かどうか
		| | +----------- 未使用
		| +------------- フレームカウンタによる割り込みの有無
		+--------------- DMCによる割り込みの有無
	*/

	is1chActive bool
	is2chActive bool
	is3chActive bool
	is4chActive bool
	is5chActive bool
	frameIRQ    bool
	dmcIRQ      bool
}

// MARK: ステータスレジスタのコンストラクタ
func NewStatusRegister() StatusRegister {
	return StatusRegister{
		is1chActive: false,
		is2chActive: false,
		is3chActive: false,
		is4chActive: false,
		is5chActive: false,
		frameIRQ:    false,
		dmcIRQ:      false,
	}
}

// MARK: フレームカウンタ割り込みの取得
func (sr *StatusRegister) FrameIRQ() bool {
	return sr.frameIRQ
}

// MARK: フレームカウンタ割り込みの設定
func (sr *StatusRegister) SetFrameIRQ(status bool) {
	sr.frameIRQ = status
}

// MARK: DMC割り込みの取得
func (sr *StatusRegister) DMCIRQ() bool {
	return sr.dmcIRQ
}

// MARK: DMC割り込みの設定
func (sr *StatusRegister) SetDMCIRQ(status bool) {
	sr.dmcIRQ = status
}

// MARK: ステータスレジスタをuint8へ変換するメソッド
func (sr *StatusRegister) ToByte() uint8 {
	var value uint8 = 0x00

	if sr.is1chActive {
		value |= 1 << STATUS_REG_IS_1CH_ACTIVE_POS
	}

	if sr.is2chActive {
		value |= 1 << STATUS_REG_IS_2CH_ACTIVE_POS
	}

	if sr.is3chActive {
		value |= 1 << STATUS_REG_IS_3CH_ACTIVE_POS
	}

	if sr.is4chActive {
		value |= 1 << STATUS_REG_IS_4CH_ACTIVE_POS
	}

	if sr.is5chActive {
		value |= 1 << STATUS_REG_IS_5CH_ACTIVE_POS
	}

	if sr.frameIRQ {
		value |= 1 << STATUS_REG_FRAME_IRQ_POS
	}

	if sr.dmcIRQ {
		value |= 1 << STATUS_REG_DMC_IRQ_POS
	}

	return value
}

// MARK: ステータスレジスタの更新メソッド
func (sr *StatusRegister) SetFromByte(value uint8) {
	sr.is1chActive = (value & (1 << STATUS_REG_IS_1CH_ACTIVE_POS)) != 0
	sr.is2chActive = (value & (1 << STATUS_REG_IS_2CH_ACTIVE_POS)) != 0
	sr.is3chActive = (value & (1 << STATUS_REG_IS_3CH_ACTIVE_POS)) != 0
	sr.is4chActive = (value & (1 << STATUS_REG_IS_4CH_ACTIVE_POS)) != 0
	sr.is5chActive = (value & (1 << STATUS_REG_IS_5CH_ACTIVE_POS)) != 0
	sr.frameIRQ = (value & (1 << STATUS_REG_FRAME_IRQ_POS)) != 0
	sr.dmcIRQ = (value & (1 << STATUS_REG_DMC_IRQ_POS)) != 0
}

// MARK: フレームカウンタの定義
type FrameCounter struct {
	disableIRQ    bool
	sequencerMode bool
}

// MARK: フレームカウンタのコンストラクタ
func NewFrameCounter() FrameCounter {
	return FrameCounter{
		disableIRQ:    false,
		sequencerMode: false,
	}
}

// MARK: フレームカウンタモードの取得
func (fc *FrameCounter) Mode() uint8 {
	if fc.sequencerMode {
		// mode = 0: 5ステップモード
		return 5
	} else {
		// mode = 1: 4ステップモード
		return 4
	}
}

// MARK: IRQ禁止フラグの取得
func (fc *FrameCounter) DisableIRQ() bool {
	return fc.disableIRQ
}

// MARK: ステータスレジスタをuint8へ変換するメソッド
func (fc *FrameCounter) ToByte() uint8 {
	var value uint8 = 0x00

	if fc.disableIRQ {
		value |= 1 << FRAME_COUNTER_IRQ_POS
	}

	if fc.sequencerMode {
		value |= 1 << FRAME_COUNTER_MODE_POS
	}

	return value
}

// MARK: フレームカウンタの更新メソッド
func (fc *FrameCounter) SetFromByte(value uint8) {
	fc.disableIRQ = (value & (1 << FRAME_COUNTER_IRQ_POS)) != 0
	fc.sequencerMode = (value & (1 << FRAME_COUNTER_MODE_POS)) != 0
}

// MARK: DMCシフトレジスタの定義
type DMCShiftRegisteer struct {
	value     uint8
	remaining uint8
}

// MARK: DMCシフトレジスタのコンストラクタ
func NewDMCShiftRegister() DMCShiftRegisteer {
	return DMCShiftRegisteer{
		value:     0x00,
		remaining: 0x08,
	}
}

// MARK: DMCシフトレジスタのシフト
func (dsr *DMCShiftRegisteer) shift() uint8 {
	value := dsr.value & 0x01

	dsr.value >>= 1

	if dsr.remaining > 0 {
		dsr.remaining--
	}

	return value
}

// MARK: DMCシフトレジスタの出力値
func (dsr *DMCShiftRegisteer) Value() uint8 {
	return dsr.value
}

// MARK: DMCシフトレジスタの出力値をセット
func (dsr *DMCShiftRegisteer) SetValue(value uint8) {
	dsr.value = value
	dsr.remaining = 8
}

// MARK: 残りビット数の取得
func (dsr *DMCShiftRegisteer) Remaining() uint8 {
	return dsr.remaining
}

// MARK: シフトレジスタが空かどうか
func (dsr *DMCShiftRegisteer) isEmpty() bool {
	return dsr.remaining == 0
}

// MARK: DMCシフトレジスタのリセット
func (dsr *DMCShiftRegisteer) reset() {
	dsr.value = 0x00
	dsr.remaining = 8
}

// MARK: ノイズシフトレジスタの定義
type NoiseShiftRegister struct {
	mode  NoiseShiftMode
	value uint16
}

// MARK: ノイズシフトレジスタのコンストラクタ
func NewNoiseShiftRegister(mode NoiseShiftMode) NoiseShiftRegister {
	return NoiseShiftRegister{
		mode:  mode,
		value: 0x0001,
	}
}

// MARK: ノイズシフトレジスタのシフト
func (nsr *NoiseShiftRegister) step() {
	/*
		短周期モード: ビット0とビット6のXOR
		長周期モード: ビット0とビット1のXOR
	*/
	var shiftBit uint16
	switch nsr.mode {
	case NOISE_MODE_LONG:
		shiftBit = 1
	case NOISE_MODE_SHORT:
		shiftBit = 6
	}

	// モードによってXORを計算
	feedback := (nsr.value & 0x01) ^ ((nsr.value >> shiftBit) & 0x01)

	// 全体を右に1ビットシフトし，ビット14に計算された値を入れる
	nsr.value >>= 1
	nsr.value = (nsr.value & 0b011_1111_1111_1111) | feedback<<14
}

// MARK: ノイズシフトレジスタのモード変更
func (nsr *NoiseShiftRegister) SetMode(mode NoiseShiftMode) {
	nsr.mode = mode
}

// MARK: ノイズシフトレジスタのビット0の値を取得
func (nsr *NoiseShiftRegister) Value() uint16 {
	// ビット0を取り出す
	return nsr.value & 0x01
}
