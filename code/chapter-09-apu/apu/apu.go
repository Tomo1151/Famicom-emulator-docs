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
)

// MARK: 変数定義
var (
	square1  = NewSquareWaveChannel()
	square2  = NewSquareWaveChannel()
	triangle = NewTriangleWaveChannel()
	noise    = NewNoiseWaveChannel()
	dmc      = NewDeltaModulationChannel()
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
}

// MARK: APUのコンストラクタ
func NewAPU() APU {
	return APU{
		channel1:     &square1,
		channel2:     &square2,
		channel3:     &triangle,
		channel4:     &noise,
		channel5:     &dmc,
		status:       NewStatusRegister(),
		frameCounter: NewFrameCounter(),
		step:         0,
		cycles:       0,
	}
}

// MARK: オーディオの初期化
func (a *APU) initAudio() {
	spec := &sdl.AudioSpec{
		Freq:     SAMPLE_RATE,
		Format:   sdl.AUDIO_F32,
		Channels: 1,
		Samples:  2048,
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

	for i := range n {
		buffer[i] = 0.0
	}
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

		a.cycles++
	}
}

// MARK: ステータスレジスタの読み込み (CPU: $4015)
func (a *APU) ReadStatus() uint8 {
	status := a.status.ToByte()
	status &= 0xF0
	// @TODO: 各チャンネルの長さカウンタ値を反映させる
	a.status.SetFrameIRQ(false) // フレームカウンタ割込みをクリア
	return status
}

// MARK: ステータスレジスタの書き込み (CPU: $4015)
func (a *APU) WriteStatus(value uint8) {
	// prev := a.status.ToByte()
	a.status.SetFromByte(value)
	// @TODO: ミュートと長さカウンタのリセット
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
}

// MARK: 線形カウンタのクロック  (3ch)
func (a *APU) tickLinearCounters() {}

// MARK: 長さカウンタのクロック (1ch / 2ch / 3ch / 4ch)
func (a *APU) tickLengthCoutners() {}

// MARK: スイープユニットのクロック (1ch / 2ch)
func (a *APU) tickSweepUnits() {}

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
