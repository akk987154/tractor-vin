package decoder

import (
	"fmt"
	"regexp"
	"strings"
)

var caseIHPattern = regexp.MustCompile(`^[A-Z0-9]{8,17}$`)

var caseIHModels = map[string]string{
	"FAR": "Farmall", "MAX": "Maxxum", "PUM": "Puma",
	"OPT": "Optum", "MAG": "Magnum", "STE": "Steiger",
	"QUA": "Quadtrac",
}

var caseIHFactories = map[string]string{
	"R": "美国威斯康星州拉辛",
	"F": "美国北达科他州法戈",
	"G": "美国内布拉斯加州格兰德艾兰",
	"B": "巴西库里蒂巴",
	"A": "奥地利圣瓦伦丁",
	"C": "中国哈尔滨",
	"I": "印度",
}

var caseYearCodes = map[string]string{
	"A": "2010", "B": "2011", "C": "2012", "D": "2013", "E": "2014",
	"F": "2015", "G": "2016", "H": "2017", "J": "2018", "K": "2019",
	"L": "2020", "M": "2021", "N": "2022", "P": "2023", "R": "2024",
	"S": "2025", "T": "2026",
}

func decodeCaseIH(serial string) (*DecodedInfo, error) {
	if !caseIHPattern.MatchString(serial) {
		return nil, fmt.Errorf("不是有效的 Case IH 序列号")
	}

	info := &DecodedInfo{
		Brand:    "Case IH",
		Metadata: make(map[string]string),
	}

	// Case IH serials often start with model designation
	for code, model := range caseIHModels {
		if strings.HasPrefix(strings.ToUpper(serial), code) {
			info.Model = model
			break
		}
	}

	if info.Model == "" {
		// Try numeric model prefix
		if len(serial) >= 3 {
			switch serial[:3] {
			case "110", "115", "120", "125", "130":
				info.Model = "Maxxum " + serial[:3]
			case "140", "145", "150", "155":
				info.Model = "Maxxum " + serial[:3]
			case "165", "175", "185", "200":
				info.Model = "Puma " + serial[:3]
			case "210", "220", "240":
				info.Model = "Puma " + serial[:3]
			case "270", "280", "300":
				info.Model = "Optum " + serial[:3]
			case "310", "340", "380":
				info.Model = "Magnum " + serial[:3]
			default:
				info.Model = "Farmall 系列"
			}
		}
	}

	// Determine series
	switch {
	case strings.Contains(info.Model, "Farmall"):
		info.Series = "Farmall Series"
		if strings.Contains(info.Model, "75") || strings.Contains(info.Model, "90") {
			info.EngineFamily = "FPT F5 2.9-3.4L"
		} else {
			info.EngineFamily = "FPT F5 3.4L"
		}
	case strings.Contains(info.Model, "Maxxum"):
		info.Series = "Maxxum Series"
		info.EngineFamily = "FPT NEF 4.5L"
	case strings.Contains(info.Model, "Puma"):
		info.Series = "Puma Series"
		info.EngineFamily = "FPT NEF 6.7L"
	case strings.Contains(info.Model, "Optum"):
		info.Series = "Optum Series"
		info.EngineFamily = "FPT Cursor 8.7L"
	case strings.Contains(info.Model, "Magnum"):
		info.Series = "Magnum Series"
		info.EngineFamily = "FPT Cursor 8.7-12.9L"
	case strings.Contains(info.Model, "Steiger") || strings.Contains(info.Model, "Quadtrac"):
		info.Series = "Steiger Series"
		info.EngineFamily = "FPT Cursor 12.9-15.9L"
	}

	// Extract factory
	if len(serial) > 6 {
		factoryChar := string(serial[6])
		if loc, ok := caseIHFactories[factoryChar]; ok {
			info.Factory = loc
		}
	}

	// Extract year
	if len(serial) > 9 {
		yearChar := string(serial[9])
		if yr, ok := caseYearCodes[yearChar]; ok {
			info.Year = yr
		}
	}

	if len(serial) > 10 {
		info.ProductionNumber = serial[10:]
	}

	if strings.Contains(info.Factory, "奥地利") || strings.Contains(info.Factory, "圣瓦伦丁") {
		info.Country = "奥地利"
	} else if strings.Contains(info.Factory, "中国") {
		info.Country = "中国"
	} else {
		info.Country = "美国"
	}

	return info, nil
}
