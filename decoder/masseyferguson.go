package decoder

import (
	"fmt"
	"regexp"
	"strings"
)

var mfPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^[A-Z0-9]{9}$`),
	regexp.MustCompile(`^[A-Z0-9]{17}$`),
}

var mfFactoryCodes = map[string]string{
	"B": "法国博韦",
	"C": "英国考文垂",
	"D": "意大利特雷维格里奥",
	"E": "美国爱荷华州",
	"F": "巴西卡诺阿斯",
	"G": "土耳其安卡拉",
	"H": "中国常州",
	"J": "印度",
}

var mfYearCodes = map[string]string{
	"A": "2010", "B": "2011", "C": "2012", "D": "2013", "E": "2014",
	"F": "2015", "G": "2016", "H": "2017", "J": "2018", "K": "2019",
	"L": "2020", "M": "2021", "N": "2022", "P": "2023", "R": "2024",
	"S": "2025", "T": "2026",
}

func decodeMasseyFerguson(serial string) (*DecodedInfo, error) {
	matched := false
	for _, pat := range mfPatterns {
		if pat.MatchString(serial) {
			matched = true
			break
		}
	}
	if !matched {
		return nil, fmt.Errorf("不是有效的 Massey Ferguson 序列号")
	}

	info := &DecodedInfo{
		Brand:    "Massey Ferguson",
		Metadata: make(map[string]string),
	}

	if len(serial) >= 4 {
		// First 2 chars often indicate model series
		prefix := serial[:2]
		switch {
		case strings.HasPrefix(prefix, "47"):
			info.Model = "4700 Series"
			info.Series = "4700 Series"
			info.EngineFamily = "AGCO Power 3.3L"
		case strings.HasPrefix(prefix, "57"):
			info.Model = "5700 Series"
			info.Series = "5700 Series"
			info.EngineFamily = "AGCO Power 4.4L"
		case strings.HasPrefix(prefix, "67"):
			info.Model = "6700S Series"
			info.Series = "6700S Series"
			info.EngineFamily = "AGCO Power 4.9L"
		case strings.HasPrefix(prefix, "77"):
			info.Model = "7700S Series"
			info.Series = "7700S Series"
			info.EngineFamily = "AGCO Power 6.6L"
		case strings.HasPrefix(prefix, "87"):
			info.Model = "8700S Series"
			info.Series = "8700S Series"
			info.EngineFamily = "AGCO Power 8.4L"
		case strings.HasPrefix(prefix, "8S"):
			info.Model = "8S Series"
			info.Series = "8S Series"
			info.EngineFamily = "AGCO Power 6.6L"
		default:
			info.Model = "MF Tractor"
		}
	}

	if len(serial) >= 6 {
		factoryCode := string(serial[5])
		if loc, ok := mfFactoryCodes[factoryCode]; ok {
			info.Factory = loc
		}
	}

	if len(serial) >= 10 {
		yearCode := string(serial[9])
		if yr, ok := mfYearCodes[yearCode]; ok {
			info.Year = yr
		}
	}

	if len(serial) > 6 {
		info.ProductionNumber = serial[6:]
	}

	info.Country = "法国/美国"
	return info, nil
}
