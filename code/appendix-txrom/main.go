package main

import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"

	"fc-emu/apu"
	"fc-emu/bus"
	"fc-emu/cartridge"
	"fc-emu/cpu"
	"fc-emu/joypad"
	"fc-emu/ppu"
)

func main() {
	// SDLの初期化
	if err := sdl.Init(sdl.INIT_VIDEO | sdl.INIT_AUDIO); err != nil {
		panic(err)
	}
	defer sdl.Quit()

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

	// コントローラの作成
	j1 := joypad.NewJoypad()
	j2 := joypad.NewJoypad()

	// APUの作成
	a := apu.NewAPU()

	// PPUの作成
	p := ppu.NewPPU(ct.Mapper(), &cv)

	// Busの作成
	b := bus.NewBus(ct.Mapper(), a, &p, &j1, &j2)

	// APUにreaderをセット
	a.SetMemoryReader(b.DMCRead)

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
				b.Shutdown()
				running = false
			case *sdl.KeyboardEvent:
				pressed := (e.State == sdl.PRESSED)

				// 1Pのキー割当て
				switch e.Keysym.Sym {
				case sdl.K_ESCAPE:
					b.Shutdown()
					running = false
				case sdl.K_w:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_UP_POS, pressed)
				case sdl.K_a:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_LEFT_POS, pressed)
				case sdl.K_s:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_DOWN_POS, pressed)
				case sdl.K_d:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_RIGHT_POS, pressed)
				case sdl.K_j:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_B_POS, pressed)
				case sdl.K_k:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_A_POS, pressed)
				case sdl.K_BACKSPACE:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_SELECT_POS, pressed)
				case sdl.K_KP_ENTER, sdl.K_RETURN:
					j1.SetButtonState(joypad.JOYPAD_BUTTON_START_POS, pressed)
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
		renderer.Present()
	}
}

// 60FPS分のティック数(高精度タイマ)を求める関数
func GetTicksPerFrame() uint64 {
	freq := uint64(sdl.GetPerformanceFrequency())
	return freq / uint64(ppu.FPS)
}
