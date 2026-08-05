package apu

import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

// MARK: 定数定義
const (
	SAMPLE_RATE        = 44_100
	APU_CYCLE_INTERVAL = 7457
	CPU_CLOCK_HZ       = 1_789_773

	// 1サンプル(44.1kHz)あたりのCPUクロック
	CYCLES_PER_SAMPLE = float64(CPU_CLOCK_HZ) / float64(SAMPLE_RATE)

	AUDIO_CHANNELS      = 1
	AUDIO_SAMPLES       = 512
	QUEUE_CHUNK_SAMPLES = 256
	MAX_QUEUED_BYTES    = SAMPLE_RATE * 4 / 10
)

// MARK: MemoryReaderの定義
type MemoryReader func(address uint16) uint8

// MARK: APUの定義
type APU struct {
	// 音声チャンネル
	channel1 *SquareWaveChannel
	channel2 *SquareWaveChannel
	channel3 *TriangleWaveChannel
	channel4 *NoiseWaveChannel
	channel5 *DeltaModulationChannel

	status       StatusRegister
	frameCounter FrameCounter
	memoryReader MemoryReader

	step   uint8   // フレームカウンタのステップ
	cycles uint    // APUサイクル
	phase  float64 // サンプル生成の位相

	buffer      []float32         // サンプルのバッファ
	audioDevice sdl.AudioDeviceID // SDLのAudioDeviceID
}

// MARK: APUのコンストラクタ
func NewAPU() *APU {
	apu := &APU{
		channel1:     NewSquareWaveChannel(),
		channel2:     NewSquareWaveChannel(),
		channel3:     NewTriangleWaveChannel(),
		channel4:     NewNoiseWaveChannel(),
		channel5:     NewDeltaModulationChannel(),
		status:       NewStatusRegister(),
		frameCounter: NewFrameCounter(),
		step:         0,
		cycles:       0,
		buffer:       []float32{},
	}

	apu.initAudio()

	return apu
}

// MARK: オーディオの初期化
func (a *APU) initAudio() {
	spec := &sdl.AudioSpec{
		Freq:     SAMPLE_RATE,
		Format:   sdl.AUDIO_F32,
		Channels: AUDIO_CHANNELS,
		Samples:  AUDIO_SAMPLES,
	}

	device, err := sdl.OpenAudioDevice("", false, spec, nil, 0)

	if err != nil {
		panic(err)
	}

	a.audioDevice = device
	a.buffer = make([]float32, 0, QUEUE_CHUNK_SAMPLES*2)

	sdl.PauseAudioDevice(a.audioDevice, false)
}

// MARK: APUのクロック
func (a *APU) Tick(cycles uint) {
	for range cycles {
		// フレームカウンタは入力の1.789MHzを7457分周する
		if a.cycles >= APU_CYCLE_INTERVAL {
			a.cycles -= APU_CYCLE_INTERVAL

			a.tickFrameCounter()
		}

		// 各チャンネルのクロック
		a.channel1.Tick()
		a.channel2.Tick()
		a.channel3.Tick()
		a.channel4.Tick()
		a.channel5.Tick()
		if a.channel5.IRQ() {
			a.status.SetDMCIRQ(true)
		}

		// CPUサイクルをサンプリングレートに変換
		if a.phase >= CYCLES_PER_SAMPLE {
			a.phase -= CYCLES_PER_SAMPLE

			// 各チャンネルの出力値をミックス
			sample := mixSamples(
				a.channel1.Output(),
				a.channel2.Output(),
				a.channel3.Output(),
				a.channel4.Output(),
				a.channel5.Output(),
			)

			// 内部バッファにサンプルを追加
			a.buffer = append(a.buffer, sample)

			// 一定数のサンプルが集まったらSDLのオーディオキューへ送信
			if len(a.buffer) >= QUEUE_CHUNK_SAMPLES {
				// キューが溢れたらクリアする
				if sdl.GetQueuedAudioSize(a.audioDevice) >= MAX_QUEUED_BYTES {
					sdl.ClearQueuedAudio(a.audioDevice)
				}

				// SDLへ送信して内部バッファを更新
				a.sendSamples(a.buffer[:QUEUE_CHUNK_SAMPLES])
				a.buffer = a.buffer[:0]
			}
		}

		// サイクルとサンプルの位相を進める
		a.cycles++
		a.phase++
	}
}

// MARK: SDLのオーディオキューへサンプルを送る
func (a *APU) sendSamples(samples []float32) {
	// サンプル数が0の時は何もしない
	if len(samples) == 0 {
		return
	}

	// []float32を[]byteに変換
	bytes := unsafe.Slice(
		(*byte)(unsafe.Pointer(&samples[0])),
		len(samples)*4,
	)

	// SDLのオーディオキューへサンプルを送信
	_ = sdl.QueueAudio(a.audioDevice, bytes)
}

// MARK: オーディオデバイスのクローズ
func (a *APU) Close() {
	// オーディオデバイスが登録済みであればクローズ
	if a.audioDevice != 0 {
		sdl.ClearQueuedAudio(a.audioDevice)
		sdl.CloseAudioDevice(a.audioDevice)
		a.audioDevice = 0
	}
}

// MARK: ステータスレジスタの読み込み (CPU: $4015)
func (a *APU) ReadStatus() uint8 {
	var status uint8

	if !a.channel1.lengthCounter.Muted() {
		status |= 1 << STATUS_REG_IS_1CH_ACTIVE_POS
	}
	if !a.channel2.lengthCounter.Muted() {
		status |= 1 << STATUS_REG_IS_2CH_ACTIVE_POS
	}
	if !a.channel3.lengthCounter.Muted() {
		status |= 1 << STATUS_REG_IS_3CH_ACTIVE_POS
	}
	if !a.channel4.lengthCounter.Muted() {
		status |= 1 << STATUS_REG_IS_4CH_ACTIVE_POS
	}
	if a.channel5.bytesLeft > 0 {
		status |= 1 << STATUS_REG_IS_5CH_ACTIVE_POS
	}
	if a.status.FrameIRQ() {
		status |= 1 << STATUS_REG_FRAME_IRQ_POS
	}
	if a.status.DMCIRQ() {
		status |= 1 << STATUS_REG_DMC_IRQ_POS
	}

	// フレームカウンタ割込みフラグをクリア
	a.status.SetFrameIRQ(false)

	return status
}

// MARK: ステータスレジスタの書き込み (CPU: $4015)
func (a *APU) WriteStatus(value uint8) {
	a.status.is1chActive = (value & (1 << STATUS_REG_IS_1CH_ACTIVE_POS)) != 0
	a.status.is2chActive = (value & (1 << STATUS_REG_IS_2CH_ACTIVE_POS)) != 0
	a.status.is3chActive = (value & (1 << STATUS_REG_IS_3CH_ACTIVE_POS)) != 0
	a.status.is4chActive = (value & (1 << STATUS_REG_IS_4CH_ACTIVE_POS)) != 0
	a.status.is5chActive = (value & (1 << STATUS_REG_IS_5CH_ACTIVE_POS)) != 0

	if !a.status.is1chActive {
		a.channel1.lengthCounter.clear()
	}
	if !a.status.is2chActive {
		a.channel2.lengthCounter.clear()
	}
	if !a.status.is3chActive {
		a.channel3.lengthCounter.clear()
	}
	if !a.status.is4chActive {
		a.channel4.lengthCounter.clear()
	}
	a.channel5.SetEnabled(a.status.is5chActive)

	// DMC割込みフラグをクリア
	a.status.SetDMCIRQ(false)
	a.channel5.SetIRQ(false)
}

// MARK: フレームカウンタの書き込み (CPU: $4017)
func (a *APU) WriteFrameCounter(value uint8) {
	a.frameCounter.SetFromByte(value)

	/*
		@NOTE: 5ステップモード時のみ$4017書き込みの副作用で half/quarterフレーム信号が発生
	*/
	if a.frameCounter.Mode() == 5 {
		a.tickEnvelopes()
		a.tickLengthCoutners()
		a.tickSweepUnits()
	}

	// 各種状態をリセット
	a.step = 0
	a.cycles = 0
	if a.frameCounter.disableIRQ {
		a.status.SetFrameIRQ(false)
	}
}

// MARK: 1chへの書き込み (矩形波)
func (a *APU) Write1ch(address uint16, value uint8) {
	a.channel1.write(address, value, a.status.is1chActive)
}

// MARK: 2chへの書き込み (矩形波)
func (a *APU) Write2ch(address uint16, value uint8) {
	a.channel2.write(address, value, a.status.is2chActive)
}

// MARK: 3chへの書き込み (三角波)
func (a *APU) Write3ch(address uint16, value uint8) {
	a.channel3.write(address, value, a.status.is3chActive)
}

// MARK: 4chへの書き込み (ノイズ)
func (a *APU) Write4ch(address uint16, value uint8) {
	a.channel4.write(address, value, a.status.is4chActive)
}

// MARK: 5chへの書き込み (DMC)
func (a *APU) Write5ch(address uint16, value uint8) {
	a.channel5.write(address, value)

	// IRQ enabledビットがクリアされるとIRQフラグも即座にクリアされる
	if address == 0x4010 && value&0x80 == 0 {
		a.status.SetDMCIRQ(false)
		a.channel5.SetIRQ(false)
	}
}

// MARK: フレームカウンタのクロック
func (a *APU) tickFrameCounter() {
	a.step++
	mode := a.frameCounter.Mode()

	switch mode {
	case 4:
		/*
			エンベロープ/線形カウンタ： e e e e   240Hz
			長さカウンタ/スイープ　　： - l - l   120Hz
			割り込み　　 　　　　　　： - - - f    60Hz
		*/
		if a.step == 1 || a.step == 2 || a.step == 3 || a.step == 4 {
			// エンベロープと線形カウンタのクロック生成 (quarter frame)
			a.tickEnvelopes()
			a.tickLinearCounters()
		}
		if a.step == 2 || a.step == 4 {
			// 長さカウンタとスイープユニットのクロック生成 (half frame)
			a.tickLengthCoutners()
			a.tickSweepUnits()
		}
		if a.step == 4 {
			// 割り込みフラグのセット
			if !a.frameCounter.DisableIRQ() {
				a.status.SetFrameIRQ(true)
			}
			a.step = 0
		}
	case 5:
		/*
			エンベロープ/線形カウンタ： e e e - e   192Hz
			長さカウンタ/スイープ　　： - l - - l    96Hz
			割り込み　　/　　　　　　： - - - - -   割り込みフラグセット無し
		*/
		if a.step == 1 || a.step == 2 || a.step == 3 || a.step == 5 {
			// エンベロープと線形カウンタのクロック生成 (quarter frame)
			a.tickEnvelopes()
			a.tickLinearCounters()
		}
		if a.step == 2 || a.step == 5 {
			// 長さカウンタとスイープユニットのクロック生成 (half frame)
			a.tickLengthCoutners()
			a.tickSweepUnits()
		}
		if a.step == 5 {
			a.step = 0
		}
	}
}

// MARK: エンベロープのクロック (1ch / 2ch / 4ch)
func (a *APU) tickEnvelopes() {
	a.channel1.envelope.Tick()
	a.channel2.envelope.Tick()
	a.channel4.envelope.Tick()
}

// MARK: 線形カウンタのクロック  (3ch)
func (a *APU) tickLinearCounters() {
	a.channel3.linearCounter.Tick(
		a.channel3.register.control,
	)
}

// MARK: 長さカウンタのクロック (1ch / 2ch / 3ch / 4ch)
func (a *APU) tickLengthCoutners() {
	a.channel1.lengthCounter.Tick()
	a.channel2.lengthCounter.Tick()
	a.channel3.lengthCounter.Tick()
	a.channel4.lengthCounter.Tick()
}

// MARK: スイープユニットのクロック (1ch / 2ch)
func (a *APU) tickSweepUnits() {
	a.channel1.sweepUnit.Tick(&a.channel1.lengthCounter, true)
	a.channel2.sweepUnit.Tick(&a.channel2.lengthCounter, false)
}

// MARK: フレームカウンタ割り込みの取得
func (a *APU) FrameIRQ() bool {
	return a.status.FrameIRQ()
}

// MARK: APU IRQの取得
func (a *APU) IRQ() bool {
	return a.status.FrameIRQ() || a.status.DMCIRQ()
}

// MARK: 各チャンネルのサンプルを適切なバランスでミックスする関数
func mixSamples(pulse1, pulse2, triangle, noise, dmc float32) float32 {
	/*
													95.88
		パルス出力 = ---------------------------
								(8128 / (1ch + 2ch)) + 100

																		159.79
		その他出力 = ---------------------------------------------------
																			1
								-------------------------------------------- + 100
								(3ch / 8227) + (4ch / 12241) + (dmc / 22638)

		最終出力 = パルス出力 + その他出力
	*/

	// 矩形波のミックス
	var pulseOut float32
	pulseSum := pulse1 + pulse2

	// 0除算の回避
	if pulseSum > 0 {
		pulseOut = 95.88 / (8128/pulseSum + 100)
	}

	// 三角波・ノイズ・DMCのミックス
	var tndOut float32
	tndSum := (triangle / 8227.0) + (noise / 12241.0) + (dmc / 22638.0)
	if tndSum > 0 {
		tndOut = 159.79 / ((1 / tndSum) + 100)
	}

	return pulseOut + tndOut
}

// MARK: MemoryReaderのセット
func (a *APU) SetMemoryReader(reader MemoryReader) {
	a.memoryReader = reader
	a.channel5.SetMemoryReader(a.memoryReader)
}
