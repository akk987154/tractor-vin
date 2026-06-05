package decoder

import (
	"fmt"
	"strings"
)

type DecodedInfo struct {
	Brand            string `json:"brand"`
	Model            string `json:"model"`
	Series           string `json:"series"`
	Year             string `json:"year"`
	Factory          string `json:"factory"`
	EngineFamily     string `json:"engineFamily"`
	ProductionNumber string `json:"productionNumber"`
	Country          string `json:"country,omitempty"`
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

func Brands() []string {
	brands := make([]string, len(decoders))
	for i, d := range decoders {
		brands[i] = d.brand
	}
	return brands
}

func Decode(serial string) (*DecodedInfo, error) {
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
