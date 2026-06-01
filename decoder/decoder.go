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

var decoders = map[string]DecoderFunc{
	"John Deere":       decodeJohnDeere,
	"Kubota":           decodeKubota,
	"Massey Ferguson":  decodeMasseyFerguson,
	"Case IH":          decodeCaseIH,
	"New Holland":      decodeNewHolland,
}

func Brands() []string {
	return []string{"John Deere", "Kubota", "Massey Ferguson", "Case IH", "New Holland"}
}

func Decode(serial string) (*DecodedInfo, error) {
	serial = strings.TrimSpace(strings.ToUpper(serial))
	if serial == "" {
		return nil, fmt.Errorf("序列号不能为空")
	}

	for brand, fn := range decoders {
		if result, err := fn(serial); err == nil {
			result.Brand = brand
			return result, nil
		}
	}

	return nil, fmt.Errorf("无法识别该序列号格式。支持的品牌: %s", strings.Join(Brands(), ", "))
}
