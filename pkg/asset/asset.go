package asset

import (
	"encoding/json"
	"log/slog"
	"strconv"
	"regexp"

	"github.com/ks-yuzu/scrape-moneyforward/pkg/metric"
)

type Asset struct {
	Category						 string		`json:"category"`
	Code								 string		`json:"code"`
	Name								 string		`json:"name"`
	Quantity						 float64	`json:"quantity"`							// 保有数
	UnitPrice						 float64	`json:"unitPrice"`						// 現在値
	AverageCost					 float64	`json:"averageCost"`					// 平均取得単価
	TotalCost						 float64	`json:"totalCost"`						// 取得価額
	Value								 float64	`json:"value"`								// 評価額
	OneDayChange				 float64	`json:"oneDayChange"`					// 前日比
	OneDayChangeRatio		 float64	`json:"oneDayChangeRatio"`		// 前日比率
	Profit							 float64	`json:"profit"`								// 評価損益
	ProfitRatio					 float64	`json:"profitRatio"`					// 評価損益率
	FinancialInstitution string		`json:"financialInstitution"` // 保有金融機関
}

func (a *Asset) Init() {
	// 可視化用に調整・計算しておく
	if a.Category == "株式（現物）" {
		if regexp.MustCompile(`^\d+$`).MatchString(a.Code) {
			a.Category = "株式（現物） - 日本"
		} else if regexp.MustCompile(`^[A-Z]+$`).MatchString(a.Code) {
			a.Category = "株式（現物） - 米国"
		}
	}

	if a.Value - a.OneDayChange != 0 {
		a.OneDayChangeRatio = a.OneDayChange / (a.Value - a.OneDayChange)
	}
}

func GenerateMetrics(assets []*Asset) string {
	labels := []map[string]string{}
	for _, a := range assets {
		labels = append(labels, map[string]string{
			"category":             a.Category,
			"code":                 a.Code,
			"name":                 a.Name,
			"quantity":							"quantity: "+strconv.FormatFloat(a.Quantity, 'f', -1, 64),
			"unitPrice":						"unitPrice: "+strconv.FormatFloat(a.UnitPrice, 'f', -1, 64),
			"averageCost":					"avgCost: "+strconv.FormatFloat(a.AverageCost, 'f', -1, 64),
			"totalCost":						"cost: "+strconv.FormatFloat(a.TotalCost, 'f', -1, 64),
			"value":								"value: "+strconv.FormatFloat(a.Value, 'f', -1, 64),
			"oneDayChange":					"change: "+strconv.FormatFloat(a.OneDayChange, 'f', -1, 64),
			"oneDayChangeRatio":		"change%: "+strconv.FormatFloat(100*a.OneDayChangeRatio, 'f', -1, 64),
			"profit":								"profit: "+strconv.FormatFloat(a.Profit, 'f', -1, 64),
			"profitRatio":					"profit%: "+strconv.FormatFloat(a.ProfitRatio, 'f', -1, 64),
			"financialInstitution": a.FinancialInstitution,
		})
	}

	result := ""

	metricFuncMap := &map[string]func(a *Asset) float64{
		"mf_asset_quantity":             func(a *Asset) float64 {return a.Quantity},
		"mf_asset_unit_price":           func(a *Asset) float64 {return a.UnitPrice},
		"mf_asset_average_cost":         func(a *Asset) float64 {return a.AverageCost},
		"mf_asset_total_cost":           func(a *Asset) float64 {return a.TotalCost},
		"mf_asset_value":                func(a *Asset) float64 {return a.Value},
		"mf_asset_one_day_change":       func(a *Asset) float64 {return a.OneDayChange},
		"mf_asset_one_day_change_ratio": func(a *Asset) float64 {return a.OneDayChangeRatio},
		"mf_asset_profit":               func(a *Asset) float64 {return a.Profit},
		"mf_asset_profit_ratio":         func(a *Asset) float64 {return a.ProfitRatio},
		// "mf_asset_allocation_weight": func(a *Asset) ...
	}
	for metricName, metricFunc := range *metricFuncMap {
		values := []float64{}
		for _, a := range assets {
			values = append(values, metricFunc(a))
		}

		result += metric.GenerateGaugeMetric(metricName, "", values, labels)
	}

	return result
}


/*
 * スクレイプしたデータから Asset オブジェクトを作るための中間データ構造
 */
type AssetMap map[string]interface{}

func (am *AssetMap) ConvertToAsset() (*Asset, error) {
	for _, key := range []string{"quantity", "unitPrice", "averageCost", "totalCost", "value", "oneDayChange", "profit", "profitRatio"} {
		(*am)[key] = toFloat((*am)[key])
	}

	jsonStr, err := json.Marshal(*am)
	if err != nil {
		return nil, err
	}

	var as Asset
	err = json.Unmarshal(jsonStr, &as)
	if err != nil {
		return nil, err
	}

	as.Init()

	return &as, nil
}

func toFloat(arg interface{}) float64 {
	switch v := arg.(type) {
	case int:
		return float64(v)
	case float64:
		return v
	case string:
		value, err := strconv.ParseFloat(
			regexp.MustCompile("[^\\-\\d\\.]+").ReplaceAllString(v, ""),
			64,
		)
		if err != nil {
			slog.Error(err.Error(), v)
			return 0
		}
		return value
	default:
		return 0
	}
}

/*
 * moneyforward の table のカラム名から Asset のフィールド名へのマッピング
 */
func ColumnName2FieldName(src string) string {
	mappingTable := map[string]string{
		"銘柄コード"					: "code",
		"種類・名称"					: "name",
		"銘柄名"							: "name",
		"名称"								: "name",
		"保有数"							: "quantity",
		"ポイント・マイル数"	: "quantity",
		"現在値"							: "unitPrice",
		"基準価額"						: "unitPrice",
		"換算レート"					: "unitPrice",
		"平均取得単価"				:	"averageCost",
		"取得価額"						: "totalCost",
		"残高"								: "value",
		"評価額"							: "value",
		"現在価値"						: "value",
		"現在の価値"					: "value",
		"前日比"							: "oneDayChange",
		"評価損益"						: "profit",
		"評価損益率"					: "profitRatio",
		"保有金融機関"				: "financialInstitution",
		"種類"								: "",
		"取得日"							: "",
		"有効期限"						: "",
		"変更"								: "",
		"削除"								: "",
	}

	dst, ok := mappingTable[src]
	if ok {
		return dst
	}

	slog.Warn("Failed to convert.", src)
	return "-"
}
