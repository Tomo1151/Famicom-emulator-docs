package main

import (
	"fmt"

	"fc-emu/cpu"
)

func main() {
	c := cpu.NewCPU()

	c.RunWithByteArray([]uint8{0xA9, 0x24, 0x29, 0x0F, 0x00})

	fmt.Println("Hello", c)
}
