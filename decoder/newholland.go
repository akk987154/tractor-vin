package decoder

import (
	"fmt"
	"regexp"
	"strings"
)

var nhPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^[A-Z0-9]{17}$`),
	regexp.MustCompile(`^[A-Z0-9]{9,13}$`),
}

var nhModels = map[string]string{
	"T4": "T4 Series", "T5": "T5 Series", "T6": "T6 Series",
	"T7": "T7 Series", "T8": "T8 Series", "T9": "T9 Series",
	"BO": "Boomer", "WK": "Workmaster",
}

var nhFactories = map[string]string{
	"A": "意大利耶西",
	"B": "英国巴西尔登",
	"C": "土耳其安卡拉",
	"D": "美国威斯康星州",
	"E": "奥地利圣瓦伦丁",
	"F": "巴西库里蒂巴",
	"G": "印度大诺伊达",
	"H": "中国哈尔滨",
	"J": "墨西哥克雷塔罗",
	"K": "波兰普沃茨克",
}

var nhYearCodes = map[string]string{
	"A": "2010", "B": "2011", "C": "2012", "D": "2013", "E": "2014",
	"F": "2015", "G": "2016", "H": "2017", "J": "2018", "K": "2019",
	"L": "2020", "M": "2021", "N": "2022", "P": "2023", "R": "2024",
	"S": "2025", "T": "2026",
}

func decodeNewHolland(serial string) (*DecodedInfo, error) {
	matched := false
	for _, pat := range nhPatterns {
		if pat.MatchString(serial) {
			matched = true
			break
		}
	}
	if !matched {
		return nil, fmt.Errorf("不是有效的 New Holland 序列号")
	}

	info := &DecodedInfo{
		Brand:    "New Holland",
		Metadata: make(map[string]string),
	}

	// New Holland serials include model prefix in some formats
	for code, model := range nhModels {
		if strings.HasPrefix(strings.ToUpper(serial), code) {
			info.Model = model
			break
		}
	}

	if info.Model == "" {
		// Check if serial contains model hint
		for i := 0; i < len(serial)-1; i++ {
			if serial[i] == 'T' && i+1 < len(serial) && serial[i+1] >= '4' && serial[i+1] <= '9' {
				info.Model = fmt.Sprintf("T%c Series", serial[i+1])
				break
			}
		}
		if info.Model == "" {
			info.Model = "T5/T6 Series"
		}
	}

	// Determine series and engine
	switch {
	case strings.Contains(info.Model, "T4"):
		info.Series = "T4 Series"
		info.EngineFamily = "FPT F5 2.9-3.4L"
	case strings.Contains(info.Model, "T5"):
		info.Series = "T5 Series"
		info.EngineFamily = "FPT NEF 4.5L"
	case strings.Contains(info.Model, "T6"):
		info.Series = "T6 Series"
		info.EngineFamily = "FPT NEF 4.5-6.7L"
	case strings.Contains(info.Model, "T7"):
		info.Series = "T7 Series"
		info.EngineFamily = "FPT NEF 6.7L"
	case strings.Contains(info.Model, "T8"):
		info.Series = "T8 Series"
		info.EngineFamily = "FPT Cursor 8.7-12.9L"
	case strings.Contains(info.Model, "T9"):
		info.Series = "T9 Series"
		info.EngineFamily = "FPT Cursor 12.9-15.9L"
	case strings.Contains(info.Model, "Boomer"):
		info.Series = "Boomer Series"
		info.EngineFamily = "LS Mtron / Shibaura 1.7-3.4L"
	case strings.Contains(info.Model, "Workmaster"):
		info.Series = "Workmaster Series"
		info.EngineFamily = "FPT F5 2.9-3.4L"
	}

	// Factory code at position 10 in VIN format
	if len(serial) >= 10 {
		factoryChar := string(serial[9])
		if loc, ok := nhFactories[factoryChar]; ok {
			info.Factory = loc
		}
	}

	if len(serial) > 10 {
		yearChar := string(serial[10])
		if yr, ok := nhYearCodes[yearChar]; ok {
			info.Year = yr
		}
	}

	if len(serial) > 11 {
		info.ProductionNumber = serial[11:]
	}

	info.Country = "意大利/美国"
	return info, nil
}
