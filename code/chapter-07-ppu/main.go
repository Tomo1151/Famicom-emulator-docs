package main

import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"

	"fc-emu/bus"
	"fc-emu/cartridge"
	"fc-emu/cpu"
	"fc-emu/ppu"
)

func main() {
	// ウィンドウの作成
	window, err := sdl.CreateWindow(
		"ファミコンエミュレータ",
		sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		int32(ppu.SCREEN_WIDTH*ppu.SCALE_FACTOR),
		int32(ppu.SCREEN_HEIGHT*ppu.SCALE_FACTOR),
		sdl.WINDOW_SHOWN,
	)
	if err != nil {
		window.Destroy()
		panic(err)
	}
	defer window.Destroy()

	// レンダラの作成
	renderer, err := sdl.CreateRenderer(
		window,
		-1,
		sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC,
	)

	if err != nil {
		window.Destroy()
		renderer.Destroy()
		panic(err)
	}

	// テクスチャの作成
	texture, err := renderer.CreateTexture(
		sdl.PIXELFORMAT_RGB24,
		sdl.TEXTUREACCESS_STREAMING,
		int32(ppu.SCREEN_WIDTH),
		int32(ppu.SCREEN_HEIGHT),
	)

	if err != nil {
		window.Destroy()
		renderer.Destroy()
		texture.Destroy()
		panic(err)
	}
	defer texture.Destroy()

	// カートリッジの作成
	ct := cartridge.NewCartridge("nestest.nes")

	// ROMファイルの読み込み
	err = ct.Load()
	if err != nil {
		panic(err)
	}

	// キャンバスの作成
	cv := ppu.NewCanvas()
	buf := unsafe.Pointer(&(cv.Buffer())[0])

	// PPUの作成
	p := ppu.NewPPU(ct.Mapper(), &cv)

	// Busの作成
	b := bus.NewBus(ct.Mapper(), &p)

	// CPUの作成
	c := cpu.NewCPU(b)

	// 初期フレーム時間の定義
	ticksPerFrame := GetTicksPerFrame()
	last := uint64(sdl.GetPerformanceCounter())
	var acc uint64

	// 描画ループ
	running := true
	for running {
		// フレーム時間計測
		now := uint64(sdl.GetPerformanceCounter())
		acc += now - last
		last = now

		// イベントループ
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent:
				if e.State == sdl.PRESSED {
					switch e.Keysym.Sym {
					case sdl.K_ESCAPE:
						running = false
					}
				}
			}
		}

		// 1フレーム分の時間が経過したとき
		for acc >= ticksPerFrame {
			// 次のPPUフレームが終わるまでCPUの実行を進める
			targetFrame := p.Frame() + 1
			for p.Frame() < targetFrame {
				c.Step()
			}
			acc -= ticksPerFrame
		}

		// テクスチャの更新
		texture.Update(nil, buf, int(ppu.SCREEN_WIDTH*3))

		// テクスチャの描画
		renderer.Clear()
		renderer.Copy(texture, nil, nil)
		renderer.Present() // 画面のリフレッシュレートに合わせて待機
	}
}

// 60FPS分のティック数(高精度タイマ)を求める関数
func GetTicksPerFrame() uint64 {
	freq := uint64(sdl.GetPerformanceFrequency())
	return freq / ppu.FPS
}
