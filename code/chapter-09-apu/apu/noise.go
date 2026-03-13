package apu

// MARK: NoiseWaveChannelの定義
type NoiseWaveChannel struct {
	register NoiseWaveRegister
	output   float32
}

// MARK: NoiseWaveChannelのコンストラクタ
func NewNoiseWaveChannel() NoiseWaveChannel {
	return NoiseWaveChannel{
		register: NewNoiseWaveRegister(),
		output:   0.0,
	}
}

// MARK: ノイズチャンネルのクロック
func (nwc *NoiseWaveChannel) Tick() {}

// MARK: ノイズチャンネルの書き込み
func (nwc *NoiseWaveChannel) write(address uint16, value uint8) {
	nwc.register.write(address, value)
}

// MARK: ノイズチャンネルの出力
func (nwc *NoiseWaveChannel) Output() float32 {
	return nwc.output
}
