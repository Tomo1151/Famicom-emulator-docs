package apu

// MARK: 変数定義
var (
	NOISE_PERIOD_TABLE = [16]uint16{
		0x004, 0x008, 0x010, 0x020, 0x040, 0x060, 0x080, 0x0A0,
		0x0CA, 0x0FE, 0x17C, 0x1FC, 0x2FA, 0x3F8, 0x7F2, 0xFE4,
	}
)

// MARK: NoiseWaveChannelの定義
type NoiseWaveChannel struct {
	register           NoiseWaveRegister
	envelope           Envelope
	lengthCounter      LengthCounter
	noiseShiftRegister NoiseShiftRegister

	timer            uint16  // タイマ
	timerPeriodIndex uint8   // チャンネルの周期
	output           float32 // 出力値
}

// MARK: NoiseWaveChannelのコンストラクタ
func NewNoiseWaveChannel() *NoiseWaveChannel {
	return &NoiseWaveChannel{
		register:           NewNoiseWaveRegister(),
		envelope:           NewEnvelope(),
		lengthCounter:      NewLengthCounter(),
		noiseShiftRegister: NewNoiseShiftRegister(NOISE_MODE_SHORT),
		timer:              0x0000,
		timerPeriodIndex:   0x00,
		output:             0.0,
	}
}

// MARK: ノイズチャンネルのクロック
func (nwc *NoiseWaveChannel) Tick() {
	// 長さカウンタが0のときはクロックしない
	if nwc.lengthCounter.Muted() {
		nwc.output = 0.0
		return
	}

	if nwc.timer > 0 {
		// クロック毎にタイマを進める
		nwc.timer--
	} else {
		// タイマが0になったらシフトレジスタを励起
		nwc.noiseShiftRegister.step()

		// テーブルから周期を取得して大麻をリセット
		nwc.timer = NOISE_PERIOD_TABLE[nwc.timerPeriodIndex] - 1
	}

	// シフトレジスタのビット0が0のときは現在のボリューム，そうでないときは0にセット
	if nwc.noiseShiftRegister.Value() == 0 {
		// 出力値をセット
		nwc.output = nwc.envelope.Volume()
	} else {
		nwc.output = 0.0
	}
}

// MARK: ノイズチャンネルの書き込み
func (nwc *NoiseWaveChannel) write(address uint16, value uint8) {
	nwc.register.write(address, value)

	switch address {
	case 0x400C:
		/*
			$400C 書き込み
			- エンベロープ
			- 長さカウンタ停止フラグ
		*/
		nwc.envelope.update(
			nwc.register.envelopePeriod,
			!nwc.register.envelopeDisable,
			nwc.register.envelopeLoop,
		)
		nwc.lengthCounter.SetHalt(nwc.register.lengthCounterHalt)
	case 0x400E:
		/*
			$400E 書き込み
			- ノイズモード
			- タイマ周期
		*/
		if nwc.register.noiseMode {
			// 生成モードフラグがセットなら短周期モード
			nwc.noiseShiftRegister.SetMode(NOISE_MODE_SHORT)
		} else {
			// 生成モードフラグがクリアなら長周期モード
			nwc.noiseShiftRegister.SetMode(NOISE_MODE_LONG)
		}
		nwc.timerPeriodIndex = nwc.register.timerPeriodIndex
	case 0x400F:
		/*
			$400F 書き込み
			- 長さカウンタ
		*/
		nwc.lengthCounter.load(nwc.register.lengthCounterLoad)

		/*
			副作用
			- エンベロープがリスタート
		*/
		nwc.envelope.reset()
	}
}

// MARK: ノイズチャンネルの出力
func (nwc *NoiseWaveChannel) Output() float32 {
	return nwc.output
}
