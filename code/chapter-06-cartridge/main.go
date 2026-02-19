package main

import (
	"fmt"

	"fc-emu/bus"
	"fc-emu/cartridge"
	"fc-emu/cpu"
)

func main() {
	// カートリッジの作成
	ct := cartridge.NewCartridge("nestest.nes")

	// ROMファイルの読み込み
	err := ct.Load()
	if err != nil {
		fmt.Println(err)
	}

	// CPUの作成・実行
	c := cpu.NewCPU(bus.NewBus(ct))
	c.Run()
}
