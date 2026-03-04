package main

import (
	"fc-emu/bus"
	"fc-emu/cartridge"
	"fc-emu/cpu"
	"fc-emu/ppu"
	"unsafe"

	"github.com/veandco/go-sdl2/sdl"
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

	// PPUの作成
	p := ppu.NewPPU(ct.Mapper(), &cv)

	// CPUの作成
	c := cpu.NewCPU(bus.NewBus(ct, &p))
	// c.Run()

	running := true
	for running {
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
