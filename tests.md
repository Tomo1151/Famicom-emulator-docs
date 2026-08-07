# テスト実施状況

## CPU

### branch_timing_tests

#### 1.Branch_Basics.nes

passed

#### 2.Backward_Branch.nes

passed

#### 3.Forward_Branch.nes

passed

### cpu_exec_space

#### test_cpu_exec_space_apu.nes

Failed #2: 4000 ERROR Mysteriously Landed at $0234 Program flow did not follow the planned path for a number of different possible reasons. Failure to obey predetermined execution path.

#### test_cpu_exec_space_ppuio.nes

Failed #3: PPU open bus implementation is missing or incomplete: A write to $2003, followed by a read from $2001 should return the same value as was written.

### cpu_interrupts_v2

#### 1-cli_latency.nes

Tests the delay in CLI taking effect, and some basic aspects of IRQ
handling and the APU frame IRQ (needed by the tests). It uses the APU's
frame IRQ and first verifies that it works well enough for the tests.

The later tests execute CLI followed by SEI and equivalent pairs of
instructions (CLI, PLP, where the PLP sets the I flag). These should
only allow at most one invocation of the IRQ handler, even if it doesn't
acknowledge the source of the IRQ. RTI is also tested, which behaves
differently. These tests also _don't_ disable interrupts after the first
IRQ, in order to test whether a pair of instructions allows only one
interrupt or causes continuous interrupts that block the main code from
continuing.

$04) Exactly one instruction after CLI should execute before IRQ is taken

#### 2-nmi_and_brk.nes

NMI behavior when it interrupts BRK. Occasionally fails on
NES due to PPU-CPU synchronization.

Result when run:
NMI BRK --
27 36 00 NMI before CLC
26 36 00 NMI after CLC
26 36 00
36 00 00 NMI interrupting BRK, with B bit set on stack
36 00 00
36 00 00
36 00 00
36 00 00
27 36 00 NMI after SEC at beginning of IRQ handler
27 36 00

NMI BRK 00
27 36 00
26 36 00
26 36 00
26 36 00
26 36 00
26 36 00
26 36 00
26 36 00
26 36 00
26 36 00

CB0122C0
Failed

#### 3-nmi_and_irq.nes

NMI behavior when it interrupts IRQ vectoring.

Result when run:
NMI IRQ
23 00 NMI occurs before LDA #1
21 00 NMI occurs after LDA #1 (Z flag clear)
21 00
20 00 NMI occurs after CLC, interrupting IRQ
20 00
20 00
20 00
20 00
20 00
20 00 Same result for 7 clocks before IRQ is vectored
25 20 IRQ occurs, then NMI occurs after SEC in IRQ handler
25 20

NMI BRK
23 00
27 23
27 23
27 23
27 23
27 23
27 23
27 23
27 23
27 23
27 23

7A096051
Failed

#### 4-irq_and_dma.nes

Has IRQ occur at various times around sprite DMA.
First column refers to what instruction IRQ occurred
after. Second column is time of IRQ, in CPU clocks relative
to some arbitrary starting point.

0 +0
1 +1
1 +2
2 +3
2 +4
4 +5
4 +6
7 +7
7 +8
7 +9
7 +10
8 +11
8 +12
8 +13
...
8 +524
8 +525
8 +526
9 +527

0 +0
0 +1
0 +2
0 +3
1 +4
1 +5
2 +6
2 +7
4 +8
4 +9
7 +10
7 +11
7 +12
7 +13
...
7 +524
7 +525
7 +526
8 +527

4670259C
Failed

#### 5-branch_delays_irq.nes

A taken non-page-crossing branch ignores IRQ during
its last clock, so that next instruction executes
before the IRQ. Other instructions would execute the
NMI before the next instruction.

The same occurs for NMI, though that's not tested here.

test_jmp
T+ CK PC
00 02 04 NOP
01 01 04
02 03 07 JMP
03 02 07
04 01 07
05 02 08 NOP
06 01 08
07 03 08 JMP
08 02 08
09 01 08

test_jmp
T+ CK PC
00 06 03
01 05 03
02 05 03
03 05 04
04 05 04
05 06 07
06 05 07
07 05 07
08 05 08
09 05 08

AD3C1F08

### cpu_reset

#### ram_after_reset.nes

passed

#### registers.nes

passed

### cpu_timing_test6/cpu_timing_test.nes

passed

### nes_instr_test

#### 01-implied.nes

passed

#### 02-immediate.nes

6B ARR #n
AB ATX #n
Failed

#### 03-zero_page.nes

passed

#### 04-zp_xy.nes

passed

#### 05-absolute.nes

passed

#### 06-abs_xy.nes

9C SYA abs,X
9E SXA abs,Y
Failed

#### 07-ind_x.nes

passed

#### 08-ind_y.nes

passed

#### 09-branches.nes

passed

#### 10-stack.nes

passed

#### 11-special.nes

passed

### instr_misc

#### 01-abs_x_wrap.nes

passed

#### 02-branch_wrap.nes

passed

#### 03-dummy_reads.nes

Tests some instructions that do dummy reads before the real read/write.
Doesn't test all instructions.

Tests LDA and STA with modes (ZP,X), (ZP),Y and ABS,X
Dummy reads for the following cases are tested:

LDA ABS,X or (ZP),Y when carry is generated from low byte
STA ABS,X or (ZP),Y
ROL ABS,X always

LDA abs,x
Failed #3

#### 04-dummy_reads_apu.nes

Tests dummy reads for (hopefully) ALL instructions which do them,
including unofficial ones. Prints opcode(s) of failed instructions.
Requires that APU implement $4015 IRQ flag reading.

1D 19 11 3D 39 31 5D 59 51 7D
79 71 9D 99 91 BD B9 B1 DD D9
D1 FD F9 F1 1E 3E 5E 7E DE FE
BC BE
Official opcodes failed
Failed #2

### instr_timng

#### 1-instr_timing.nes

13 was 9, should be 8 (cross)
1B was 8, should be 7 (cross)
1F was 8, should be 7 (cross)
33 was 9, should be 8 (cross)
3B was 8, should be 7 (cross)
3F was 8, should be 7 (cross)
53 was 9, should be 8 (cross)
5B was 8, should be 7 (cross)
5F was 8, should be 7 (cross)
73 was 9, should be 8 (cross)
7B was 8, should be 7 (cross)
7F was 8, should be 7 (cross)
93 was 0, should be 6
9F was 0, should be 5
D3 was 9, should be 8 (cross)
DB was 8, should be 7 (cross)
DF was 8, should be 7 (cross)
F3 was 9, should be 8 (cross)
FB was 8, should be 7 (cross)
FF was 8, should be 7 (cross)

Unofficial instruction timing is wrong
Failed #4

#### 2-branch_timing.nes

passed

## PPU

### color_test.nes

passed

### blargg_ppu_tests_2005.09.15b

#### palette_ram.nes

$01) passed

#### power_up_palette.nes

Reports whether initial values in palette at power-up match those
that my NES has. These values are probably unique to my NES.

$02) Palette differs from table

#### sprite_ram.nes

$01) passed

#### vbl_clear_time.nes

$01) passed

#### vram_access.nes

$01) passed

### full_nes_palette.nes

passed

### nmi_sync / demo_ntsc.nes

きれいな逆コの字になっている  
\*\*\*\*\*\*\*\*\*  
　　　　　 　 \*\*\*  
\*\*\*\*\*\*\*\*\*

### ntsc_torture.nes

passed

### oam_read.nes

passed

### oam_stress.nes

59916E5B failed
最初のアスタリスク 1 つが-になっている，それ以外はきれいに表示

### oam3.nes

NES 2.0 format isn't supported

### palette.nes

NES 2.0 format isn't supported

### ppu_open_bus.nes

Failed #2
Write to any PPU register should set decay value

### ppu_read_buffer.nes

Unsupported mapper type: 03

### ppu_vbl_nmi.nes

Unsupported mapper type: 01

### scrolltest/scroll.nes

Unsupported mapper type: 01

### sprite_hit_tests_2005.10.05

#### 01.basics.nes

passed

#### 02.alignment.nes

passed

#### 03.corners.nes

passed

#### 04.flip.nes

passed

#### 05.left_clip.nes

passed

#### 06.right_edge.nes

passed

#### 07.screen_bottom.nes

passed

#### 08.double_height.nes

passed

#### 09.timing_basics.nes

passed

#### 10.timing_order.nes

passed

#### 11.edge_timing.nes

passed

### sprite_overflow_tests

#### 1.Basics.nes

passed

#### 2.Details.nes

passed

#### 3.Timing.nes

Tests timing of sprite overflow flag. The tests fail if timing is off by
more than a CPU clock or two.

5. Set too late for first scanline

#### 4.Obscure.nes

Tests the pathological behavior when 8 sprites are on a scanline and the
one just after the 8th is not on the scanline. In that case, the PPU
interprets different bytes of each following sprite as the Y coordinate.
For the following setup of any consecutive range of sprites (that is,
sprite 1 below could be the PPU's 25th sprite, sprite 2 the 26th, etc.):

    1 2 3 4 5 6 7 8 9 10 11 12 13 14

If 1-8 are on the same scanline but 9 isn't, then the second byte of 10,
the third byte of 11, fourth byte of 12, first byte of 13, second byte
of 14, etc. are treated as those sprites' Y coordinates for the purpose
of determining whether overflow occurs on that scanline. This search
continues until one of the (erroneously interpreted) Y coordinates
places the sprite within the scanline, or all sprites have been scanned.
Refer to the NESdevWiki for further information about this behavior.

Failed #2
$02) Checks that second byte of sprite #10 is treated as its Y

#### 5.Emulator.nes

passed

### spritecans-2011/spritecans.nes

passed

### stars_se/StarsSE.NES

passed

### vbl_nmi_timing

#### 1.frame_basics.nes

passed

#### 2.vbl_timing.nes

Tests timing of VBL being set, and special case where reading VBL flag
as it would be set causes it to not be set for that frame.

Failed #4 4) Flag should read as clear 2 PPU clocks before VBL

#### 3.even_odd_frame.nes

Test clock skipped when BG is enabled on odd PPU frames. Tests
enable/disable BG during 5 consecutive frames, then see how many clocks
were skipped. Patterns are shown as XXXXX, where each X can either be B
(BG enabled) or - (BG disabled).

Failed #2 2) Pattern ----- should not skip any clocks

#### 4.vbl_clear_timing.nes

passed

#### 5.nmi_suppression.nes

Tests timing of NMI suppression when reading VBL flag just as it's set,
and that this doesn't occur when reading one clock before or after.

Failed #3
$03) Reading flag when it's set should suppress NMI

#### 6.nmi_disable.nes

passed

#### 7.nmi_timing.nes

passed

## APU

### apu_mixer

#### square.nes

passed

#### triangle.nes

passed

#### noise.nes

failed

#### dmc.nes

failed

### apu_reset

#### 4015_cleared.nes

passed

#### 4017_timing.nes

Delay after effective $4017
write: 14

Frame IRQ flag should be set later after power/reset
Failed #2

#### 4017_written.nes

At reset, $4017 should should be rewritten with last value written
Failed #3

#### irq_flag_cleared.nes

passed

#### len_ctrs_enabled.nes

passed

#### works_immediately.nes

passed

### apu_test

#### 1-len_ctr.nes

passed

#### 2-len_table.nes

passed

#### 3-irq_flag.nes

passed

#### 4-jitter.nes

Tests for APU clock jitter. Also tests basic timing of frame irq flag
since it's needed to determine jitter.

Frame irq is set too soon
Faield #2

#### 5-len_timing.nes

Verifies timing of length counter clocks in both modes

Channel: 0
First length of mode 0 is too soon
Failed #2

#### 6-irq_flag_timing.nes

Frame interrupt flag is set three times in a row 29831 clocks after
writing $00 to $4017.

Flag first set too soon
Failed #2

#### 7-dmc_basics.nes

Verifies basic DMC operation

There should be a one-byte buffer that's filled immediately if empty
Failed #19

#### 8-dmc_rates.nes

passed
