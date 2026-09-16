package decoder

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var jdPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^1[A-Z]{2}\d{4}[A-Z]\d{6}$`),
	regexp.MustCompile(`^LV[A-Z]{4}\d[A-Z]\d{4}$`),
	regexp.MustCompile(`^[A-Z]{2}\d{4}[A-Z]\d{6}$`),
	regexp.MustCompile(`^1RW\d{4}[A-Z]\d{6}$`),
	regexp.MustCompile(`^[A-Z0-9]{13}$`),
	regexp.MustCompile(`^[A-Z0-9]{17}$`),
}

var jdFactoryCodes = map[string]string{
	"A": "美国乔治亚州奥古斯塔",
	"B": "美国爱荷华州滑铁卢",
	"C": "美国爱荷华州达文波特",
	"D": "美国伊利诺伊州莫林",
	"E": "美国爱荷华州滑铁卢发动机厂",
	"F": "美国威斯康星州霍里孔",
	"G": "德国曼海姆",
	"H": "德国茨韦布吕肯",
	"J": "墨西哥萨尔蒂约",
	"K": "巴西蒙特内格鲁",
	"L": "印度普纳",
	"M": "美国明尼苏达州",
	"N": "中国天津",
	"P": "美国爱荷华州安克尼",
	"R": "美国北达科他州法戈",
	"S": "美国堪萨斯州科菲维尔",
	"T": "美国俄亥俄州",
	"W": "德国曼海姆发动机厂",
}

var jdYearCodes = map[string]string{
	"A": "2010", "B": "2011", "C": "2012", "D": "2013", "E": "2014",
	"F": "2015", "G": "2016", "H": "2017", "J": "2018", "K": "2019",
	"L": "2020", "M": "2021", "N": "2022", "P": "2023", "R": "2024",
	"S": "2025", "T": "2026",
}

var jdModelFromSerial = map[string]string{
	"6030": "6030", "4044": "4044M", "4066": "4066R",
	"5045": "5045E", "5055": "5055E", "5065": "5065E", "5075": "5075E",
	"5080": "5080E", "5085": "5085M", "5090": "5090M", "5100": "5100M",
	"5105": "5105M", "6110": "6110M", "6125": "6125M", "6130": "6130M",
	"6135": "6135M", "6140": "6140M", "6145": "6145M", "6155": "6155M",
	"6170": "6170M", "6175": "6175M", "6195": "6195M",
	"7210": "7210R", "7230": "7230R", "7250": "7250R", "7270": "7270R",
	"7290": "7290R", "7310": "7310R", "7330": "7330R",
	"8370": "8370R", "8400": "8400R", "9420": "9420RX", "9470": "9470RX",
	"3038": "3038E", "3046": "3046R",
}

func decodeJohnDeere(serial string) (*DecodedInfo, error) {
	// 这里必须严格按 jdPatterns 判定。
	// 曾经存在一个"部分匹配"兜底分支：只要 len(serial) >= 5 且以 "1" 开头就放行，
	// 而 John Deere 排在 decoders 列表的第一位，于是任何以 "1" 开头的输入
	// （包括 "1ZZZZ" 这种无意义字符串）都会被判定为 John Deere 并直接返回，
	// 后面的 Kubota / Massey Ferguson / New Holland / Case IH 永远没有机会执行。
	if !matchesAny(jdPatterns, serial) {
		return nil, fmt.Errorf("不是有效的 John Deere PIN 码")
	}

	info := &DecodedInfo{
		Brand:    "John Deere",
		Metadata: make(map[string]string),
	}

	// Parse factory code from 13/17 char PIN
	// Format: 1RW8136P#####
	if len(serial) >= 8 {
		if len(serial) >= 3 {
			info.Metadata["WMI"] = serial[0:3]
			country := map[string]string{"1RW": "美国", "LVR": "德国", "ZJD": "巴西"}
			if c, ok := country[serial[0:3]]; ok {
				info.Country = c
			}
		}

		if len(serial) >= 8 {
			factoryCode := string(serial[7])
			if loc, ok := jdFactoryCodes[factoryCode]; ok {
				info.Factory = loc
			} else {
				info.Factory = fmt.Sprintf("工厂代码 %s", factoryCode)
			}
		}

		if len(serial) >= 10 {
			yearCode := string(serial[9])
			if yr, ok := jdYearCodes[yearCode]; ok {
				info.Year = yr
			}
		}

		if len(serial) > 10 {
			info.ProductionNumber = serial[10:]
		}
	}

	// Try to determine model from serial prefix
	if len(serial) < 6 {
		return info, nil
	}
	modelSerial := serial[2:6]
	if len(modelSerial) == 4 {
		if m, ok := jdModelFromSerial[modelSerial]; ok {
			info.Model = m
			// Determine series from model
			if strings.Contains(m, "E") || strings.Contains(m, "M") {
				if n, err := strconv.Atoi(modelSerial[:2]); err == nil {
					info.Series = fmt.Sprintf("%d Series", n/10*10)
				}
			} else if strings.Contains(m, "R") {
				info.Series = fmt.Sprintf("%sR Series", modelSerial[:1])
			}
		} else {
			info.Model = fmt.Sprintf("%s系列", modelSerial)
		}
	}

	// Determine engine family based on model number
	if strings.Contains(info.Model, "3") {
		info.EngineFamily = "PowerTech E 1.6-2.9L"
	} else if strings.Contains(info.Model, "5") {
		info.EngineFamily = "PowerTech E 2.9-4.5L"
	} else if strings.Contains(info.Model, "6") {
		info.EngineFamily = "PowerTech PSS 4.5-6.8L"
	} else if strings.Contains(info.Model, "7") || strings.Contains(info.Model, "8") {
		info.EngineFamily = "PowerTech PSS 6.8-9.0L"
	} else if strings.Contains(info.Model, "9") {
		info.EngineFamily = "PowerTech PSS 13.5-15.0L"
	} else {
		info.EngineFamily = "PowerTech 系列"
	}

	return info, nil
}
