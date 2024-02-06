package main

import (
	"fmt"
	"log/slog"
	"errors"
	"context"
	"time"
	"os"
	"strings"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
	"github.com/PuerkitoBio/goquery"

	"github.com/ks-yuzu/scrape-moneyforward/pkg/asset"
)

var (
	LOGIN_EMAIL    = getEnv("LOGIN_EMAIL", "")
	LOGIN_PASSWORD = getEnv("LOGIN_PASSWORD", "")
)

func main() {
	err := run()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}

func run() error {
	bytes, err := os.ReadFile(mustGetEnv("PORTFOLIO_HTML"))
	if err != nil {
		return err
	}

	portfolio, err := parsePortfolioHtml(string(bytes))
	if len(portfolio) == 0 {
		return errors.New("Found no assets.")
	}
	fmt.Println(asset.GenerateMetrics(portfolio))

	return nil


	if (LOGIN_EMAIL == "") {
		return errors.New("LOGIN_EMAIL must be set")
	}
	if (LOGIN_PASSWORD == "") {
		return errors.New("LOGIN_PASSWORD must be set")
	}

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.DisableGPU,
		// chromedp.NoSandbox,
		// chromedp.Headless,
		chromedp.Flag("disable-infobars", true),
		chromedp.UserDataDir(getEnv("USER_DATA_DIR", "/tmp/scrape-moneyforward")),
	}...)
	defer cancel()

	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(func(format string, args ...any) {
			slog.Default().Info(fmt.Sprintf(format, args...))
		}),
	)
	defer cancel()

	// ctx, cancel = context.WithTimeout(ctx, 10 * time.Minute)
	// defer cancel()

	err = chromedp.Run(ctx, chromedp.ActionFunc(scrape))
	if err != nil {
		return err
	}

	return nil
}


func scrape(ctx context.Context) error {
	PORTFOLIO_URL := "https://moneyforward.com/bs/portfolio"
	LOGIN_URL     := "https://id.moneyforward.com/sign_in"

	chromedp.Navigate(PORTFOLIO_URL).Do(ctx)
	chromedp.WaitVisible("html", chromedp.ByQuery).Do(ctx)

	var currentUrl string
	chromedp.Location(&currentUrl).Do(ctx)

	switch {
	case strings.HasPrefix(currentUrl, PORTFOLIO_URL):
		break
	case strings.HasPrefix(currentUrl, LOGIN_URL):
		var nodes []*cdp.Node
		if chromedp.Nodes(`input[id="mfid_user[email]"]`, &nodes, chromedp.AtLeast(0)).Do(ctx); len(nodes) > 0 {
			chromedp.SetValue(`input[id="mfid_user[email]"]`, LOGIN_EMAIL).Do(ctx)
			chromedp.Submit(`form[action="/sign_in/email"] #submitto`).Do(ctx)
			chromedp.WaitVisible("body", chromedp.ByQuery).Do(ctx)
		}
		if chromedp.Nodes(`input[id="mfid_user[password]"]`, &nodes, chromedp.AtLeast(0)).Do(ctx); len(nodes) > 0 {
			chromedp.SetValue(`input[id="mfid_user[password]"]`, LOGIN_PASSWORD).Do(ctx)
			chromedp.Submit(`form[action="/sign_in"] #submitto`).Do(ctx)
			chromedp.WaitVisible("body", chromedp.ByQuery).Do(ctx)
		}
	default:
		return fmt.Errorf("Unsupported redirect to %s", currentUrl)
	}

	if chromedp.Location(&currentUrl).Do(ctx); !strings.HasPrefix(currentUrl, PORTFOLIO_URL) {
		return fmt.Errorf("Failed to navigate to %s", PORTFOLIO_URL)
	}

	slog.Debug("Succeeded in opening portfolio page")

	var html string
	chromedp.OuterHTML("html", &html, chromedp.NodeVisible, chromedp.ByQuery).Do(ctx)
	slog.Debug("html: ", html)

	chromedp.Sleep(30 * time.Minute).Do(ctx)

	return nil
}


func parsePortfolioHtml(html string) ([]*asset.Asset, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	// overviewTable := doc.Find("table").First
	// overviewTable.Find("tr").Each(func(i int, s *goquery.Selection) {
	// 	cells := s.Find("th, td").Map(func(i int, s *goquery.Selection) string {
	// 		return strings.TrimSpace(s.Text())
	// 	})
	// 	fmt.Printf("%+v\n", cells)
	// })

	portfolio := []*asset.Asset{}

	sections := doc.Find(`section[id^="portfolio_det"]`)
	sections.Each(func(i int, s *goquery.Selection) {
		categoryName := strings.TrimSpace(s.Find("h1").First().Text())
		slog.Debug("category:", categoryName)

		s.Find("table").Each(func(i int, s *goquery.Selection) {
			fieldNames := s.Find("tr:first-of-type > th").Map(func(i int, s *goquery.Selection) string {
				return asset.ColumnName2FieldName(strings.TrimSpace(s.Text()))
			})

			s.Find("tr").Slice(1, goquery.ToEnd).Each(func(i int, s *goquery.Selection) {
				am := &asset.AssetMap{"category": categoryName}
				s.Find("td").Each(func(i int, s *goquery.Selection) {
					(*am)[fieldNames[i]] = strings.TrimSpace(s.Text())
				})

				// データの事前調整
				if categoryName == "株式（信用）" { // 信用取引の場合は, 損益=評価額
					(*am)["value"] = (*am)["profit"]
				}

				as, err := am.ConvertToAsset()
				if err != nil {
					fmt.Printf("%+v\nskip %+v\n", err, am)
					return
				}

				portfolio = append(portfolio, as)
			})
		})
	})

	return portfolio, nil
}

func getEnv(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}

func mustGetEnv(key string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}

	panic(errors.New("failed to get env: "+key))
}
