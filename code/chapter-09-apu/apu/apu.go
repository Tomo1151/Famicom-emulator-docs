package apu

/*
#include <stdint.h>
void AudioCallback(void* userdata, uint8_t* stream, int length);
*/
import "C"
import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
)

// MARK: 定数定義
const (
	APU_CYCLE_INTERVAL = 7457
	SAMPLE_RATE        = 44_100
	CPU_CLOCK_HZ       = 1_789_773
	BUFFER_SIZE        = 2048
)

const CYCLES_PER_SAMPLE = float64(CPU_CLOCK_HZ) / float64(SAMPLE_RATE)

// MARK: 変数定義
var (
	apu *APU
)

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

	step   uint8 // フレームカウンタのステップ
	cycles uint
	phase  float64

	buffer *RingBuffer // サンプルのバッファ
}

// MARK: APUのコンストラクタ
func NewAPU() *APU {
	ch1 := NewSquareWaveChannel()
	ch2 := NewSquareWaveChannel()
	ch3 := NewTriangleWaveChannel()
	ch4 := NewNoiseWaveChannel()
	ch5 := NewDeltaModulationChannel()

	a := &APU{
		channel1:     &ch1,
		channel2:     &ch2,
		channel3:     &ch3,
		channel4:     &ch4,
		channel5:     &ch5,
		status:       NewStatusRegister(),
		frameCounter: NewFrameCounter(),
		step:         0,
		cycles:       0,
		buffer:       NewRingBuffer(),
	}

	apu = a

	a.initAudio()

	return a
}

// MARK: オーディオの初期化
func (a *APU) initAudio() {
	spec := &sdl.AudioSpec{
		Freq:     SAMPLE_RATE,
		Format:   sdl.AUDIO_F32,
		Channels: 1,
		Samples:  BUFFER_SIZE / 2,
		Callback: sdl.AudioCallback(C.AudioCallback),
	}

	if err := sdl.OpenAudio(spec, nil); err != nil {
		panic(err)
	}

	// オーディオ再生開始
	sdl.PauseAudio(false)
}

// MARK: SDLのオーディオコールバック
//
//export AudioCallback
func AudioCallback(userdata unsafe.Pointer, stream *C.uint8_t, length C.int) {
	n := int(length) / 4
	buffer := unsafe.Slice((*float32)(unsafe.Pointer(stream)), n)

	if apu == nil {
		for i := range n {
			buffer[i] = 0.0
		}
		return
	}

	apu.buffer.Read(buffer)
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

		// CPUサイクルをサンプリングレートに変換
		if a.phase >= CYCLES_PER_SAMPLE {
			a.phase -= CYCLES_PER_SAMPLE

			// サンプリングレートに合わせて出力値をバッファへ書き込み
			a.buffer.Write(
				mixSamples(
					a.channel1.Output(),
					a.channel2.Output(),
					a.channel3.Output(),
					a.channel4.Output(),
					a.channel5.Output(),
				),
			)
		}

		// サイクルとサンプルの位相を進める
		a.cycles++
		a.phase++
	}
}

// MARK: ステータスレジスタの読み込み (CPU: $4015)
func (a *APU) ReadStatus() uint8 {
	var status uint8

	if a.channel1.lengthCounter.Value() > 0 {
		status |= 1 << STATUS_REG_IS_1CH_ACTIVE_POS
	}
	if a.channel2.lengthCounter.Value() > 0 {
		status |= 1 << STATUS_REG_IS_2CH_ACTIVE_POS
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

	// DMC割込みフラグをクリア
	a.status.SetDMCIRQ(false)
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
	a.status.SetFrameIRQ(false)
}

// MARK: 1chへの書き込み (矩形波)
func (a *APU) Write1ch(address uint16, value uint8) {
	a.channel1.write(address, value)
}

// MARK: 2chへの書き込み (矩形波)
func (a *APU) Write2ch(address uint16, value uint8) {
	a.channel2.write(address, value)
}

// MARK: 3chへの書き込み (三角波)
func (a *APU) Write3ch(address uint16, value uint8) {
	a.channel3.write(address, value)
}

// MARK: 4chへの書き込み (ノイズ)
func (a *APU) Write4ch(address uint16, value uint8) {
	a.channel4.write(address, value)
}

// MARK: 5chへの書き込み (DMC)
func (a *APU) Write5ch(address uint16, value uint8) {
	a.channel5.write(address, value)
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
}

// MARK: 線形カウンタのクロック  (3ch)
func (a *APU) tickLinearCounters() {}

// MARK: 長さカウンタのクロック (1ch / 2ch / 3ch / 4ch)
func (a *APU) tickLengthCoutners() {
	a.channel1.lengthCounter.Tick()
	a.channel2.lengthCounter.Tick()
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
		tndOut = 159.79 / (1/tndSum + 100)
	}

	return pulseOut + tndOut
}
