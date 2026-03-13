package apu

// MARK: TriangleWaveChannelの定義
type TriangleWaveChannel struct {
	register TriangleWaveRegister
	output   float32
}

// MARK: TriangleWaveChannelのコンストラクタ
func NewTriangleWaveChannel() TriangleWaveChannel {
	return TriangleWaveChannel{
		register: NewTriangleWaveRegister(),
		output:   0.0,
	}
}

// MARK: 三角波チャンネルのクロック
func (twc *TriangleWaveChannel) Tick() {}

// MARK: 三角波チャンネルの書き込み
func (twc *TriangleWaveChannel) write(address uint16, value uint8) {
	twc.register.write(address, value)
}

// MARK: 三角波チャンネルの出力
func (twc *TriangleWaveChannel) Output() float32 {
	return twc.output
}
