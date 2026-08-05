package apu

// MARK: 変数定義
var (
	SQUARE_DUTY_TABLE = [4][8]uint8{
		{0, 1, 0, 0, 0, 0, 0, 0}, // 12.5%
		{0, 1, 1, 0, 0, 0, 0, 0}, // 25.0%
		{0, 1, 1, 1, 1, 0, 0, 0}, // 50.0%
		{1, 0, 0, 1, 1, 1, 1, 1}, // 75.0%
	}
)

// MARK: SquareWaveChannelの定義
type SquareWaveChannel struct {
	register      SquareWaveRegister
	envelope      Envelope
	lengthCounter LengthCounter
	sweepUnit     SweepUnit

	duty      uint8   // デューティ比
	timer     uint16  // タイマ
	sequencer uint    // シーケンサ
	output    float32 // 出力値
}

// MARK: SquareWaveChannelのコンストラクタ
func NewSquareWaveChannel() *SquareWaveChannel {
	return &SquareWaveChannel{
		register:      NewSquareWaveRegister(),
		envelope:      NewEnvelope(),
		lengthCounter: NewLengthCounter(),
		sweepUnit:     NewSweepUnit(),
		duty:          0x00,
		timer:         0x0000,
		sequencer:     0x00,
		output:        0.0,
	}
}

// MARK: 矩形波チャンネルのクロック
func (swc *SquareWaveChannel) Tick() {
	// 長さカウンタまたはスイープユニットが0のときはクロックしない
	if swc.lengthCounter.Muted() || swc.sweepUnit.Muted() {
		swc.output = 0.0
		return
	}

	// チャンネルの周期を取得
	period := swc.sweepUnit.Period()

	if swc.timer > 0 {
		// クロック毎にタイマを進める
		swc.timer--
	} else {
		// タイマが0になったらシーケンサを進める
		swc.sequencer = (swc.sequencer + 1) & 0x07

		// 矩形波タイマは2CPUクロック毎に進むため，周期を2倍して合わせる
		swc.timer = (period+1)*2 - 1
	}

	// デューティ比シーケンステーブルから，1のときは現在のボリューム，そうでないときは0にセット
	if SQUARE_DUTY_TABLE[swc.duty][swc.sequencer] == 1 {
		// 出力値をセット
		swc.output = swc.envelope.Volume()
	} else {
		swc.output = 0.0
	}
}

// MARK: 矩形波チャンネルの書き込み
func (swc *SquareWaveChannel) write(address uint16, value uint8, isActive bool) {
	swc.register.write(address, value)

	switch address {
	case 0x4000, 0x4004:
		/*
			$4000 / $4004 書き込み
			- デューティ比
			- エンベロープ
			- 長さカウンタ停止フラグ

			デューティ比は即座に更新されるが，シーケンサの位置は影響を受けない
		*/
		swc.duty = swc.register.duty
		swc.envelope.update(
			swc.register.envelopePeriod,
			!swc.register.envelopeDisable,
			swc.register.envelopeLoop,
		)
		swc.lengthCounter.SetHalt(swc.register.lengthCounterHalt)
	case 0x4001, 0x4005:
		/*
			$4001 / $4005 書き込み
			- スイープユニット
		*/
		swc.sweepUnit.update(
			swc.register.sweepShift,
			swc.register.sweepDirection,
			swc.register.sweepPeriod,
			swc.register.sweepEnabled,
		)
	case 0x4002, 0x4006:
		/*
			$4002 / $4006 書き込み
			- タイマ下位8ビット
		*/
		swc.sweepUnit.timerPeriod = (swc.sweepUnit.timerPeriod & 0xFF00) | uint16(swc.register.timerLower)
	case 0x4003, 0x4007:
		/*
			$4003 / $4007 書き込み
			- タイマ上位3ビット
			- 長さカウンタ
		*/
		swc.sweepUnit.timerPeriod = uint16(swc.register.timerUpper)<<8 | (swc.sweepUnit.timerPeriod & 0x00FF)
		if isActive {
			swc.lengthCounter.load(swc.register.lengthCounterLoad)
		}

		/*
			副作用
			- シーケンサが即座にリセット
			- エンベロープがリスタート (分周器の周期はリセットされない)
		*/
		swc.sequencer = 0
		swc.envelope.reset()
	}
}

// MARK: 矩形波チャンネルの出力
func (swc *SquareWaveChannel) Output() float32 {
	return swc.output
}
