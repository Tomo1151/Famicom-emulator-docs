package apu

// MARK: 変数定義
var (
	TRIANGLE_SEQUENCE_TABLE = [16 * 2]uint8{
		15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0,
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	}
)

// MARK: TriangleWaveChannelの定義
type TriangleWaveChannel struct {
	register      TriangleWaveRegister
	lengthCounter LengthCounter
	linearCounter LinearCounter

	timer       uint16  // タイマ
	timerPeriod uint16  // チャンネルの周期
	sequencer   uint    // シーケンサ
	output      float32 // 出力値
}

// MARK: TriangleWaveChannelのコンストラクタ
func NewTriangleWaveChannel() *TriangleWaveChannel {
	return &TriangleWaveChannel{
		register:      NewTriangleWaveRegister(),
		lengthCounter: NewLengthCounter(),
		linearCounter: NewLinearCounter(),
		timer:         0x0000,
		sequencer:     0x00,
		output:        0.0,
	}
}

// MARK: 三角波チャンネルのクロック
func (twc *TriangleWaveChannel) Tick() {
	// 長さカウンタまたは線形カウンタが0の時はクロックしない
	if twc.lengthCounter.Muted() || twc.linearCounter.Muted() {
		twc.output = 0.0
		return
	}

	if twc.timer > 0 {
		// クロック毎にタイマを進める
		twc.timer--
	} else {
		// タイマが0になったらシーケンサを進める
		twc.sequencer = (twc.sequencer + 1) & 0x1F

		// チャンネルの周期でタイマをリセット
		twc.timer = twc.timerPeriod
	}

	// 出力値を正規化してセット
	twc.output = float32(TRIANGLE_SEQUENCE_TABLE[twc.sequencer]) / 15.0
}

// MARK: 三角波チャンネルの書き込み
func (twc *TriangleWaveChannel) write(address uint16, value uint8) {
	twc.register.write(address, value)

	switch address {
	case 0x4008:
		/*
			$4008 書き込み
			- 線形カウンタ
			- 長さカウンタ停止フラグ
		*/
		twc.linearCounter.update(
			twc.register.linearCounterReload,
			twc.register.linearCounterControl,
		)
		twc.lengthCounter.SetHalt(twc.register.lengthCounterHalt)
	case 0x400A:
		/*
			$400A 書き込み
			- タイマ下位8ビット
		*/
		twc.timerPeriod = (twc.timerPeriod & 0xFF00) | uint16(twc.register.timerLower)
	case 0x400B:
		/*
			$400B 書き込み
			- タイマ上位3ビット
			- 長さカウンタ
		*/
		twc.timerPeriod = uint16(twc.register.timerUpper)<<8 | (twc.timerPeriod & 0x00FF)
		twc.lengthCounter.load(twc.register.lengthCounterLoad)

		/*
			副作用
			- 線形カウンタのコントロールフラグがセット
		*/
		twc.linearCounter.SetReload()
	}
}

// MARK: 三角波チャンネルの出力
func (twc *TriangleWaveChannel) Output() float32 {
	return twc.output
}
