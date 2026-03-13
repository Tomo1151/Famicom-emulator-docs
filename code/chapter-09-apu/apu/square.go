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

	duty        uint8
	timerReload uint16
	timer       uint16
	sequencer   uint8

	output float32
}

// MARK: SquareWaveChannelのコンストラクタ
func NewSquareWaveChannel() SquareWaveChannel {
	return SquareWaveChannel{
		register: NewSquareWaveRegister(),
		output:   0.0,
	}
}

// MARK: 矩形波チャンネルのクロック
func (swc *SquareWaveChannel) Tick() {
	frequency := swc.sweepUnit.Frequency()
	if frequency < 8 || frequency > 0x7FFF || swc.lengthCounter.Muted() || swc.sweepUnit.Muted() {
		swc.output = 0.0
		return
	}

	swc.timerReload = (uint16(frequency) + 1) * 2
	if swc.timer == 0 {
		swc.timer = swc.timerReload
	}
}

// MARK: 矩形波チャンネルの書き込み
func (swc *SquareWaveChannel) write(address uint16, value uint8) {
	swc.register.write(address, value)
}

// MARK: 矩形波チャンネルの出力
func (swc *SquareWaveChannel) Output() float32 {
	return swc.output
}
