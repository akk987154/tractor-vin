package decoder

import (
	"fmt"
	"regexp"
	"strings"
)

// MaxSerialLength 是序列号允许的最大长度（字节）。
// VIN 标准为 17 位，这里留出余量以兼容各厂商更短的序列号格式。
// 上限的存在是为了给正则匹配和 ToUpper 复制划定边界：
// 没有上限时，一个超大字符串会先被完整复制一份，再逐个跑完所有品牌的正则。
const MaxSerialLength = 32

type DecodedInfo struct {
	Brand            string            `json:"brand"`
	Model            string            `json:"model"`
	Series           string            `json:"series"`
	Year             string            `json:"year"`
	Factory          string            `json:"factory"`
	EngineFamily     string            `json:"engineFamily"`
	ProductionNumber string            `json:"productionNumber"`
	Country          string            `json:"country,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
}

type DecoderFunc func(serial string) (*DecodedInfo, error)

type decoderEntry struct {
	brand string
	fn    DecoderFunc
}

// decoders is an ordered slice (not a map) to guarantee deterministic brand
// selection. Case IH has an extremely permissive pattern, so it must always
// be tried last to avoid swallowing serials that belong to other brands.
var decoders = []decoderEntry{
	{"John Deere", decodeJohnDeere},
	{"Kubota", decodeKubota},
	{"Massey Ferguson", decodeMasseyFerguson},
	{"New Holland", decodeNewHolland},
	{"Case IH", decodeCaseIH},
}

// matchesAny 报告 serial 是否命中任一 pattern。
// 注意：各品牌的模式之间存在大面积重叠（例如 `^[A-Z0-9]{17}$` 同时属于
// John Deere、Massey Ferguson、New Holland，且是 Case IH `^[A-Z0-9]{8,17}$`
// 的子集），因此品牌判定实际上完全取决于 decoders 的顺序，而不是模式本身的区分度。
func matchesAny(patterns []*regexp.Regexp, serial string) bool {
	for _, p := range patterns {
		if p.MatchString(serial) {
			return true
		}
	}
	return false
}

func Brands() []string {
	brands := make([]string, len(decoders))
	for i, d := range decoders {
		brands[i] = d.brand
	}
	return brands
}

func Decode(serial string) (*DecodedInfo, error) {
	if len(serial) > MaxSerialLength {
		return nil, fmt.Errorf("序列号长度不能超过 %d 个字符", MaxSerialLength)
	}

	serial = strings.TrimSpace(strings.ToUpper(serial))
	if serial == "" {
		return nil, fmt.Errorf("序列号不能为空")
	}

	for _, d := range decoders {
		if result, err := d.fn(serial); err == nil {
			result.Brand = d.brand
			return result, nil
		}
	}

	return nil, fmt.Errorf("无法识别该序列号格式。支持的品牌: %s", strings.Join(Brands(), ", "))
}
