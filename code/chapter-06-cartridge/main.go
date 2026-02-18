package main

import (
	"fc-emu/cpu"
)

func main() {
	c := cpu.NewCPU()

	// 配列から以下のプログラムを実行
	// LDA #$24    ; A = $24
	// AND #$0F    ; A = A & $0F
	// BRK         ; break0
	c.RunWithByteArray([]uint8{0xA9, 0x24, 0x29, 0x0F, 0x00})
}
