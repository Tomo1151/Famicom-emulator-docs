package main

import (
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"

	"fc-emu/bus"
	"fc-emu/cartridge"
	"fc-emu/cpu"
	"fc-emu/joypad"
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

	// コントローラの作成
	j1 := joypad.NewJoypad()
	j2 := joypad.NewJoypad()

	// PPUの作成
	p := ppu.NewPPU(ct.Mapper(), &cv)

	// Busの作成
	b := bus.NewBus(ct, &p, &j1, &j2)

	// CPUの作成
	c := cpu.NewCPU(b)

	// 描画ループ
	running := true
	for running {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.KeyboardEvent:
				pressed := (e.State == sdl.PRESSED)

				// 1Pのキー割当て
				switch e.Keysym.Sym {
				case sdl.K_ESCAPE:
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

		// 次のPPUフレームが終わるまでCPUの実行を進める
		tagetFrame := p.Frame() + 1
		for p.Frame() < tagetFrame {
			c.Step()
		}

		// テクスチャの更新
		texture.Update(nil, unsafe.Pointer(&(cv.Buffer())[0]), int(ppu.SCREEN_WIDTH*3))

		// テクスチャの描画
		renderer.Clear()
		renderer.Copy(texture, nil, nil)
		renderer.Present()
	}
}
