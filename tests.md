# テスト実施状況

## CPU

## PPU

### color_test.nes

実装をしていない色強調以外は問題なし

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

Tests PPU VRAM read/write and internal read buffer operation

$06) Palette read should also read VRAM into read buffer

### full_nes_palette.nes

青が4色くらいで四分割くらいになっている
色強調はまだ未実装

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
最初のアスタリスク1つが-になっている，それ以外はきれいに表示

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

Tests sprite 0 hit with regard to bottom of screen.

Failed #3
$03) Can hit when Y < 239

#### 08.double_height.nes

passed

#### 09.timing_basics.nes

Tests sprite 0 hit timing to within 12 or so PPU clocks. Tests flag
timing for upper-left corner, upper-right corner, lower-right corner,
and time flag is cleared (at end of VBL). Depends on proper PPU frame
length (less than 29781 CPU clocks).

Failed #3
$03) Upper-left corner too late

#### 10.timing_order.nes

Tests sprite 0 hit timing for which pixel it first reports hit on. Each
test hits at the same location on screen, though different relative to
the position of the sprite.

Failed #3
$03) Upper-left corner too late

#### 11.edge_timing.nes

passed

### sprite_overflow_tests

#### 1.Basics.nes

Tests basic operation of sprite overflow flag.

Failed #6
$6) Shouldn't be set when all rendering is off

#### 2.Details.nes

passed

#### 3.Timing.nes

Tests timing of sprite overflow flag. The tests fail if timing is off by
more than a CPU clock or two.

Failed #2
$02) Cleared too late

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

Tests things that an emulator with predictive overflow flag handling is
likely to get wrong.

Failed #3
$03) Disabling rendering didn't recalculate flag time

### spritecans-2011/spritecans.nes

passed

### stars_se/StarsSE.NES

passed

### vbl_nmi_timing

#### 1.frame_basics.nes

Tests basic VBL flag operation and general timing of PPU frames.

Failed #5
$05) PPU frame with BG enabled is too long

#### 2.vbl_timing.nes

Tests timing of VBL being set, and special case where reading VBL flag
as it would be set causes it to not be set for that frame.

Failed #8
$08) Reading 1 PPU clock before VBL should suppress setting

#### 3.even_odd_frame.nes

Test clock skipped when BG is enabled on odd PPU frames. Tests
enable/disable BG during 5 consecutive frames, then see how many clocks
were skipped. Patterns are shown as XXXXX, where each X can either be B
(BG enabled) or - (BG disabled).

Failed #3
$03) Pattern BB--- should skip 1 clock

#### 4.vbl_clear_timing.nes

Tests timing of VBL flag clearing.

passed

#### 5.nmi_suppression.nes

Tests timing of NMI suppression when reading VBL flag just as it's set,
and that this doesn't occur when reading one clock before or after.

Failed #3
$03) Reading flag when it's set should suppress NMI

#### 6.nmi_disable.nes

Tests NMI occurrence when disabling NMI just as VBL flag is set, and
just after.

Failed #2
$02) NMI should occur when disabled 3 PPU clocks after VBL

#### 7.nmi_timing.nes

Tests timing of NMI and immediate occurrence when enabled with VBL flag
already set.

$03) NMI occurred 2 PPU clocks too early

## APU
