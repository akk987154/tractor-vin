package decoder

import (
	"strings"
	"testing"
)

// 回归测试：John Deere 曾经有一个"部分匹配"兜底分支 ——
// `if len(serial) >= 5 && strings.HasPrefix(serial, "1")` 就直接放行。
// 由于 John Deere 排在 decoders 列表首位，任何以 "1" 开头的输入都会被
// 判定为 John Deere 并立即返回，后面的 Kubota / Massey Ferguson /
// New Holland / Case IH 永远没有机会执行。
func TestDecodeDoesNotFallBackToJohnDeere(t *testing.T) {
	// 这些输入都不匹配任何 John Deere 模式，因此必须整体解码失败
	for _, serial := range []string{"1ZZZZ", "1ABCD", "1XY", "1!!!!"} {
		info, err := Decode(serial)
		if err == nil {
			t.Errorf("Decode(%q) 本应失败，实际得到 brand=%q model=%q",
				serial, info.Brand, info.Model)
		}
	}
}

// 兜底分支的典型症状：成功返回但所有字段都是空的。
// 这种"200 但无内容"的响应对下游（tractor-compare / tractor-log）毫无意义。
func TestDecodeNeverReturnsEmptySuccess(t *testing.T) {
	inputs := []string{
		"1ZZZZ", "1ABCD", "L3560ABC", strings.Repeat("1", 17),
		strings.Repeat("A", 13), "MX6000AB", "T5.100", "8S305",
	}
	for _, serial := range inputs {
		info, err := Decode(serial)
		if err != nil {
			continue
		}
		if info.Brand == "" {
			t.Errorf("Decode(%q) 返回成功但没有品牌", serial)
		}
		if info.Model == "" && info.Series == "" && info.Year == "" &&
			info.Factory == "" && info.ProductionNumber == "" {
			t.Errorf("Decode(%q) 返回成功但所有字段均为空（疑似部分匹配兜底）", serial)
		}
	}
}

// 回归测试：kubotaModelPrefix 原本是 map，而其中的键互为前缀
// （"L" 是 "L35"/"L39"/"L47"/"L60" 的前缀；"M" 是 "M5"/"M6"/"M7"/"MX" 的前缀；
// "B" 是 "BX" 的前缀）。map 迭代顺序随机，同一个序列号在不同请求中
// 可能解出不同的型号。改用有序切片后结果必须完全稳定。
func TestDecodeIsDeterministic(t *testing.T) {
	inputs := []string{"L3560ABC", "L4701XY", "M7060AB", "MX6000", "BX2380", "M5091"}

	for _, serial := range inputs {
		first, err := Decode(serial)
		if err != nil {
			t.Fatalf("Decode(%q) 失败: %v", serial, err)
		}
		for i := 0; i < 300; i++ {
			got, err := Decode(serial)
			if err != nil {
				t.Fatalf("Decode(%q) 第 %d 次调用失败: %v", serial, i, err)
			}
			if got.Brand != first.Brand || got.Model != first.Model || got.Series != first.Series {
				t.Fatalf("Decode(%q) 结果不稳定：第 %d 次得到 %q/%q/%q，首次为 %q/%q/%q",
					serial, i, got.Brand, got.Model, got.Series,
					first.Brand, first.Model, first.Series)
			}
		}
	}
}

// 最长前缀优先：更具体的前缀必须赢过更宽泛的前缀
func TestKubotaPrefersLongestPrefix(t *testing.T) {
	cases := map[string]string{
		"L3560ABC": "L35",
		"L6060XY":  "L60",
		"M5091":    "M5", // "M" 与 "M5" 都是前缀，应命中更长的
		"MX6000":   "MX", // 而不是 "M"
		"BX2380":   "BX", // 而不是 "B"
		"L4701":    "L47",
	}

	for serial, wantModel := range cases {
		info, err := Decode(serial)
		if err != nil {
			t.Fatalf("Decode(%q) 失败: %v", serial, err)
		}
		if info.Model != wantModel {
			t.Errorf("Decode(%q).Model = %q，期望 %q", serial, info.Model, wantModel)
		}
	}
}

// 断言 kubotaPrefixes 的排序不变式：若 A 比 B 长且 B 是 A 的前缀，
// 则 A 必须排在 B 前面，否则最长前缀优先会失效。
func TestKubotaPrefixTableOrdering(t *testing.T) {
	for i, a := range kubotaPrefixes {
		for j, b := range kubotaPrefixes {
			if i == j || len(a.prefix) <= len(b.prefix) {
				continue
			}
			if strings.HasPrefix(a.prefix, b.prefix) && i > j {
				t.Errorf("%q（更长且以 %q 为前缀）排在第 %d 位，却在 %q 的第 %d 位之后",
					a.prefix, b.prefix, i, b.prefix, j)
			}
		}
	}
}

func TestDecodeRejectsEmptyAndWhitespace(t *testing.T) {
	for _, serial := range []string{"", " ", "   ", "\t", "\n", "  \t \n "} {
		if _, err := Decode(serial); err == nil {
			t.Errorf("Decode(%q) 应返回错误", serial)
		}
	}
}

// 长度上限既在 API 层也在解码器层生效，避免超大输入被完整复制后再逐条跑正则
func TestDecodeRejectsOverlongSerial(t *testing.T) {
	_, err := Decode(strings.Repeat("1", MaxSerialLength+1))
	if err == nil {
		t.Fatalf("长度 %d 的序列号应被拒绝", MaxSerialLength+1)
	}
	if !strings.Contains(err.Error(), "长度") {
		t.Errorf("超长序列号应因长度被拒绝，实际错误: %v", err)
	}

	// 恰好等于上限时不应触发长度错误（能否解码取决于格式本身）
	if _, err := Decode(strings.Repeat("1", MaxSerialLength)); err != nil &&
		strings.Contains(err.Error(), "长度") {
		t.Errorf("长度恰好为上限 %d 时不应报长度错误: %v", MaxSerialLength, err)
	}
}

func TestDecodeIsCaseInsensitive(t *testing.T) {
	lower, errLower := Decode("l3560abc")
	upper, errUpper := Decode("L3560ABC")
	if errLower != nil || errUpper != nil {
		t.Fatalf("大小写输入应都能解码: %v / %v", errLower, errUpper)
	}
	if lower.Brand != upper.Brand || lower.Model != upper.Model || lower.Series != upper.Series {
		t.Errorf("大小写结果不一致: %+v vs %+v", lower, upper)
	}
}

// 任何输入都不应触发 panic。
// CHANGELOG 记录过一次切片越界（serial[2:6] 在 len(serial) >= 8 的守卫之外），
// 这里用一组短输入与畸形输入把各分支扫一遍。
func TestDecodeNeverPanics(t *testing.T) {
	inputs := []string{
		"", " ", "1", "1A", "1AB", "1ABC", "12", "123", "1234", "12345",
		"1ZZZZ", "-1", "null", "北京", "🚜", "L", "M", "B", "0",
		strings.Repeat("A", 17), strings.Repeat("1", 17), strings.Repeat("0", 13),
		"1RW8136P123456", "LVRA1234B1234", "1AB1234C123456",
	}

	for _, serial := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Decode(%q) 触发 panic: %v", serial, r)
				}
			}()
			// 结果本身不作断言，只要求不 panic
			_, _ = Decode(serial)
		}()
	}
}

func TestBrandsMatchesDecoders(t *testing.T) {
	brands := Brands()
	if len(brands) != len(decoders) {
		t.Fatalf("Brands() 返回 %d 项，decoders 有 %d 项", len(brands), len(decoders))
	}
	// Case IH 的模式是 ^[A-Z0-9]{8,17}$，覆盖面最广，必须排在最后
	if last := brands[len(brands)-1]; last != "Case IH" {
		t.Errorf("最后一个品牌应为 Case IH，实际为 %q", last)
	}
	for _, b := range brands {
		if b == "" {
			t.Error("存在空品牌名")
		}
	}
}
