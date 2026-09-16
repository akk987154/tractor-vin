package decoder

import (
	"fmt"
	"regexp"
	"strings"
)

var kubotaPattern = regexp.MustCompile(`^[A-Z]{0,2}\d{4,6}[A-Z]{0,3}\d{0,5}$`)

// kubotaPrefixEntry 描述一个型号前缀及其对应的系列/型号标识。
type kubotaPrefixEntry struct {
	prefix string
	series string
	model  string
}

// kubotaPrefixes 按 prefix 长度从长到短排列，必须保持这个顺序。
//
// 注意：这些前缀之间存在包含关系 —— "L" 是 "L35"/"L39"/"L47"/"L60" 的前缀，
// "M" 是 "M5"/"M6"/"M7"/"MX" 的前缀，"B" 是 "BX" 的前缀。
// 原实现是 `for prefix, model := range kubotaModelPrefix`，map 迭代顺序随机，
// 于是同一个序列号可能这次解出 "L Series (紧凑型)"、下次解出 "L3560"。
// 这正是 CHANGELOG 里声称已在 decoder.go 修好的那类非确定性 bug，
// 当时只改了品牌派发，漏掉了品牌文件内部的这一处。
// 改用有序切片 + 最长前缀优先，结果稳定且优先命中更具体的型号。
var kubotaPrefixes = []kubotaPrefixEntry{
	{"L35", "L3560", "L35"},
	{"L39", "L3901", "L39"},
	{"L47", "L4701", "L47"},
	{"L60", "L6060", "L60"},
	{"M5", "M5 Series", "M5"},
	{"M6", "M6 Series", "M6"},
	{"M7", "M7 Series", "M7"},
	{"MX", "MX Series", "MX"},
	{"BX", "BX Series (庭院型)", "BX"},
	{"L", "L Series (紧凑型)", "L"},
	{"M", "M Series (多用途)", "M"},
	{"B", "B Series (超紧凑型)", "B"},
}

var kubotaYearPrefix = map[string]string{
	"1": "1990-1995", "2": "1995-2000", "3": "2000-2005",
	"4": "2005-2010", "5": "2010-2015", "6": "2015-2020",
	"7": "2020-2023", "8": "2023-2025", "9": "2025+",
}

var kubotaFactories = map[string]string{
	"J": "日本大阪堺市",
	"T": "美国乔治亚州盖恩斯维尔",
	"F": "法国比尔叙瓦勒",
	"K": "韩国大邱",
	"C": "中国无锡",
}

func decodeKubota(serial string) (*DecodedInfo, error) {
	if !kubotaPattern.MatchString(serial) {
		return nil, fmt.Errorf("不是有效的 Kubota 序列号")
	}

	info := &DecodedInfo{
		Brand:    "Kubota",
		Metadata: make(map[string]string),
	}

	// Try to extract model from prefix（最长前缀优先，见 kubotaPrefixes 的说明）
	serial = strings.TrimSpace(serial)
	for _, entry := range kubotaPrefixes {
		if strings.HasPrefix(serial, entry.prefix) {
			info.Series = entry.series
			info.Model = entry.model
			break
		}
	}

	if info.Model == "" {
		// Extract model from alphanumeric prefix
		for i, c := range serial {
			if c >= '0' && c <= '9' {
				if i > 0 {
					info.Model = serial[:i]
				}
				break
			}
		}
		if info.Model == "" {
			info.Model = "未知型号"
		}
	}

	// Year from numeric part
	info.ProductionNumber = serial
	for _, c := range serial {
		if c >= '0' && c <= '9' {
			firstDigit := string(c)
			if yr, ok := kubotaYearPrefix[firstDigit]; ok {
				info.Year = yr
			}
			break
		}
	}

	// Check for factory code in suffix
	if len(serial) > 4 {
		lastChar := strings.ToUpper(string(serial[len(serial)-1]))
		if loc, ok := kubotaFactories[lastChar]; ok {
			info.Factory = loc
		} else {
			info.Factory = "日本大阪 (默认)"
		}
	}

	// Determine engine family
	if strings.HasPrefix(info.Model, "L") {
		info.EngineFamily = "Kubota 03 Series (1.5-3.8L)"
	} else if strings.HasPrefix(info.Model, "M") {
		if strings.Contains(info.Model, "7") || strings.Contains(info.Model, "8") {
			info.EngineFamily = "Kubota V6108 (6.1L)"
		} else if strings.Contains(info.Model, "6") {
			info.EngineFamily = "Kubota V3800 (3.8L)"
		} else {
			info.EngineFamily = "Kubota V3300/V3800 (3.3-3.8L)"
		}
	} else if strings.HasPrefix(info.Model, "B") {
		info.EngineFamily = "Kubota D902 (0.9L) / D1105 (1.1L)"
	}

	info.Country = "日本"
	return info, nil
}
