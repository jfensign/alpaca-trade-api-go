package main

import (
	"fmt"
	"math"
	"os"
	_ "sort"
	"time"
    _ "unicode"
	"github.com/alpacahq/alpaca-trade-api-go/polygon"
	"github.com/alpacahq/alpaca-trade-api-go/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/common"
	"github.com/shopspring/decimal"
	"github.com/VividCortex/ewma"
)

const (
	layoutISO = "2006-01-02"
)

type clientWrapper struct {
	alpaca      *alpaca.Client
	polygon     *polygon.Client
	ma         map[string]ewma.MovingAverage
	ma2        map[string]ewma.MovingAverage
	ma3        map[string]ewma.MovingAverage
	ema         map[string]ewma.MovingAverage
	ema2        map[string]ewma.MovingAverage
	ema3        map[string]ewma.MovingAverage
	mma         map[string]ewma.MovingAverage
	mma2        map[string]ewma.MovingAverage
	mma3        map[string]ewma.MovingAverage
	currPrice   map[string]float64
	tickOpen      map[string]float64
	percentChange map[string]float64
	eq            map[string]float64
	portfolioShare map[string]float64
	targetPositionValue map[string]float64
	momentum  map[string]float64
	momentum2 map[string]float64
	momentum3 map[string]float64
	amountToAdd map[string]float64
	positionVal map[string]float64
	tema        map[string]float64
	lastOrderID map[string]string
	positionQty   map[string]int
	accountGains  []float64
	allTickers  []string
	blackList   []string
	period     int
	period2    int
	period3    int
	portfolioVal float64
	buyList  []string
	sellList []string
	buyingPower float64
	// runningAverage float64
	// lastOrder      string
	// amtBars        int
	// stock          string
	// blacklist []string
}



var (
	clientContainer *clientWrapper
	defaultUniverse []string
	marketsOpen     bool
)

func init() {
	API_KEY := ""
	API_SECRET := ""
	// Check for environment variables
	if common.Credentials().ID == "" {
		os.Setenv(common.EnvApiKeyID, API_KEY)
	}
	if common.Credentials().Secret == "" {
		os.Setenv(common.EnvApiSecretKey, API_SECRET)
	}

	clientContainer = &clientWrapper{
		alpaca.NewClient(common.Credentials()),
		polygon.NewClient(common.Credentials()),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]ewma.MovingAverage),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]float64),
		make(map[string]string),
		make(map[string]int),
		[]float64{},
		[]string{},
		[]string{},
		20,
		30,
		50,
		4000.00,
		[]string{},
		[]string{},
		0.0,
	}

	account, err := clientContainer.alpaca.GetAccount()
    if err != nil {
    	fmt.Println(err)
    }
	clientContainer.portfolioVal, _ = account.Cash.Float64()
	clientContainer.buyingPower, _  = account.BuyingPower.Float64()
	positions, _ := clientContainer.alpaca.ListPositions()
	for _, p := range positions {
		rawVal, _ := p.MarketValue.Float64()
		clientContainer.portfolioVal += rawVal
	}
}

func main() {
	marketsOpen = false
	list := []string{"ACST", "ONCY", "GFI", "OGEN", "MYOV", "ISSC", "QGEN", "DVAX", "NLTX", "SUPV", "GRTX", "APLT", "SPPI", "VALU", "AU", "HSDT", "PTI", "TSLA", "PD", "NEWR", "LCTX", "HCR", "TRXC", "OGEN", "MYOV", "NGD", "SILV", "QGEN", "SPPI", "BBAR", "DRRX", "ITCI", "NEM", "RAD", "AMRN", "CLVS", "INTT", "MRAM", "BIMI", "LK", "AKBA", "NVAX", "CCL", "MBOT", "PTI", "CLPS", "SLNO", "SAVA", "ENTX", "SSNT", "FSM", "ELSE", "RUBY"}
    for _, t := range list {
      clientContainer.AddTickerToStrategy(t)
    }
    for {
		clock, err := clientContainer.alpaca.GetClock()
		if err != nil {
			panic(err)
		}
		if !clock.IsOpen {
			marketsOpen = true
			clientContainer.RunStrategy()
			if clock.NextClose.Sub(clock.Timestamp) < 15*time.Minute {
				clientContainer.closeAllPositions()
			}
		}else{
			if marketsOpen {
				fmt.Println("Market Closed. Will resume when they next open. Closing positions.")
				clientContainer.closeAllPositions()
			}
			marketsOpen = false
		}
		time.Sleep(1 * time.Minute)
	}
}

func (g *clientWrapper) AddTickerToStrategy(ticker string) {
	for _, i := range g.allTickers {
		if i == ticker {
			return
		}
	}

	g.allTickers = append(g.allTickers, ticker)
}

func (g *clientWrapper) RefreshUniverse() {
	watchlists, err := g.alpaca.GetWatchLists()
	if  err != nil {
		panic(err)
	}
	for _, w := range watchlists {
		for _, a := range w.Assets {
			g.AddTickerToStrategy(a.Symbol)
		}
	}
}

func (g *clientWrapper) Model(ticker string) {
	account, err := clientContainer.alpaca.GetAccount()
    if err != nil {
      return
    }
	clientContainer.portfolioVal, _ = account.Cash.Float64()
	clientContainer.buyingPower, _  = account.BuyingPower.Float64()
	positions, _ := g.alpaca.ListPositions()
	for _, p := range positions {
		rawVal, _ := p.MarketValue.Float64()
		g.portfolioVal += rawVal
	}
	// Get current position
	position, err := clientContainer.alpaca.GetPosition(ticker)
	if err != nil {
	  // Do nothing. Client returns an error if a position does not exist.
	}else{
	  g.positionQty[ticker] = int(position.Qty.IntPart())
	  g.positionVal[ticker], _ = position.MarketValue.Float64()
	}
	// Current time
	currTime     := time.Now()
	// currTimeStr  := currTime.Format(layoutISO)
	startTime    := currTime.AddDate(0, 0, -21)
	// start        := startTime.Format(layoutISO)
	// Get the new updated price and running average.
	bars, _ := g.alpaca.GetSymbolBars(ticker, alpaca.ListBarParams{
		Timeframe: "day",
		Limit: &g.period,
		StartDt: &startTime,
	})

	bars2, _ := g.alpaca.GetSymbolBars(ticker, alpaca.ListBarParams{
		Timeframe: "day",
		Limit: &g.period2,
		StartDt: &startTime,
	})

	bars3, _ := g.alpaca.GetSymbolBars(ticker, alpaca.ListBarParams{
		Timeframe: "day",
		Limit: &g.period3,
		StartDt: &startTime,
	})

	// Reset Moving Averages
	g.ma[ticker]  = ewma.NewMovingAverage()
	g.ma2[ticker]  = ewma.NewMovingAverage()
	g.ma3[ticker]  = ewma.NewMovingAverage()
	g.ema[ticker] = ewma.NewMovingAverage(float64(len(bars)))
	g.ema2[ticker] = ewma.NewMovingAverage(float64(len(bars2)))
	g.ema3[ticker] = ewma.NewMovingAverage(float64(len(bars3)))
	g.mma[ticker] = ewma.NewMovingAverage(float64(len(bars)))
	g.mma2[ticker] = ewma.NewMovingAverage(float64(len(bars2)))
	g.mma3[ticker] = ewma.NewMovingAverage(float64(len(bars3)))
	g.momentum[ticker]  = float64(g.currPrice[ticker] - float64(bars[0].Close))
	g.momentum2[ticker] = float64(g.currPrice[ticker] - float64(bars2[0].Close))
	g.momentum3[ticker] = float64(g.currPrice[ticker] - float64(bars3[0].Close))
	if len(bars) == 0 {
		fmt.Println("No Bars for ", ticker)
		return
	}
	// Update Current Price
	g.tickOpen[ticker]       = float64(bars[0].Open)
	// Update Current Price
	g.currPrice[ticker]      = float64(bars[len(bars)-1].Close)
    // Update percent change
	g.percentChange[ticker]  = (g.currPrice[ticker] - g.tickOpen[ticker]) / g.tickOpen[ticker]

	for tick, bar := range bars {
		g.ma[ticker].Add(float64(bar.Close))
		g.ema[ticker].Add(float64(bar.Close))
        
        if tick == 0 {
        	g.mma[ticker].Add(float64(bar.Close))
        }else {
        	g.mma[ticker].Add(float64(bar.Close - bars[tick-1].Close))
        }
	}

	for tick, bar := range bars2 {
		g.ma2[ticker].Add(float64(bar.Close))
		g.ema2[ticker].Add(float64(bar.Close))
        
        if tick == 0 {
        	g.mma2[ticker].Add(float64(bar.Close))
        }else {
        	g.mma2[ticker].Add(float64(bar.Close - bars2[tick-1].Close))
        }
	}

	for tick, bar := range bars3 {
		g.ma3[ticker].Add(float64(bar.Close))
		g.ema3[ticker].Add(float64(bar.Close))
        
        if tick == 0 {
        	g.mma3[ticker].Add(float64(bar.Close))
        }else {
        	g.mma3[ticker].Add(float64(bar.Close - bars3[tick-1].Close))
        }
	}

    g.tema[ticker] = (3 * g.ema[ticker].Value()) - (3 * g.ema2[ticker].Value()) + g.ema3[ticker].Value()
	g.portfolioShare[ticker] = (g.ma[ticker].Value() - g.currPrice[ticker]) / g.currPrice[ticker] * 20
	g.targetPositionValue[ticker] = g.portfolioVal * g.portfolioShare[ticker]
	g.amountToAdd[ticker] = g.targetPositionValue[ticker] - g.positionVal[ticker]
}

func (g *clientWrapper) Rebalance(ticker string) {
	if g.ma[ticker] == nil || g.ema == nil {
		return
	}
	if g.currPrice[ticker] > g.tema[ticker] {
		if g.ma[ticker].Value() < g.tema[ticker] {
			// Sell our position if the price is above the running average, if any.
			if g.positionQty[ticker] > 0 {
				g.submitLimitOrder(g.positionQty[ticker], ticker, g.currPrice[ticker], "sell")
				g.sellList = append(g.sellList, ticker)
				g.LogSale(ticker, "sell")
			}
		}
	}else{
		if g.ma[ticker].Value() > g.tema[ticker] {
			// Add to our position, constrained by our buying power; or, sell down to optimal amount of shares.
			if g.amountToAdd[ticker] > 0 {
				if g.amountToAdd[ticker] > g.buyingPower {
					g.amountToAdd[ticker] = g.buyingPower
				}
				var qtyToBuy = int(g.amountToAdd[ticker] / g.currPrice[ticker])
				g.submitLimitOrder(qtyToBuy, ticker, g.currPrice[ticker], "buy")

			} else {
				g.amountToAdd[ticker] *= -1
				var qtyToSell = int(g.amountToAdd[ticker] / g.currPrice[ticker])
				if qtyToSell > g.positionQty[ticker] {
					qtyToSell = g.positionQty[ticker]
				}
				g.submitLimitOrder(qtyToSell, ticker, g.currPrice[ticker], "buy")
			}
			g.buyList = append(g.buyList, ticker)
			g.LogSale(ticker, "buy")
		}else{
			g.LogSale(ticker, "pass")
		}
	}
}

func (g *clientWrapper) ClearOrders() {
	clientContainer.sellList = []string{}
	clientContainer.buyList  = []string{}
	for _, t := range g.allTickers {
		if g.lastOrderID[t] != "" {
          _ = g.alpaca.CancelOrder(g.lastOrderID[t])
		}
	}
}

func (g *clientWrapper) closeAllPositions() {
	positions, _ := g.alpaca.ListPositions()
	for _, position := range positions {
		var orderSide string
		if position.Side == "long" {
			orderSide = "sell"
		} else {
			orderSide = "buy"
		}
		qty, _ := position.Qty.Float64()
		qty = math.Abs(qty)
		g.submitMarketOrder(int(qty), position.Symbol, orderSide)
	}
}

func (g *clientWrapper) RunStrategy() {
	g.RefreshUniverse()
	g.ClearOrders()
	for _, ticker := range g.allTickers {
		g.Model(ticker)
		g.Rebalance(ticker)
	}
	fmt.Printf("Buys: %v\nSells: %v\n", clientContainer.buyList, clientContainer.sellList)
}

// Submit a limit order if quantity is above 0.
func (g *clientWrapper) submitLimitOrder(qty int, ticker string, price float64, side string) error {
	if qty > 0 {
		account, err := clientContainer.alpaca.GetAccount()
		if err != nil {
			return err
		}
		adjSide := alpaca.Side(side)
		limPrice := decimal.NewFromFloat(price)
		order, err := g.alpaca.PlaceOrder(alpaca.PlaceOrderRequest{
			AccountID:   account.ID,
			AssetKey:    &ticker,
			Qty:         decimal.NewFromFloat(float64(qty)),
			Side:        adjSide,
			Type:        "limit",
			LimitPrice:  &limPrice,
			TimeInForce: "day",
		})
		fmt.Printf("Limit order of | %d %s %s | sent.\n", qty, ticker, side)
		if err == nil {
			fmt.Printf("Limit order of | %d %s %s | sent.\n", qty, ticker, side)
			g.lastOrderID[ticker] = order.ID
		}
		return err
	}
	return nil
}

// Submit a market order if quantity is above 0.
func (g *clientWrapper) submitMarketOrder(qty int, ticker string, side string) error {
	if qty > 0 {
		account, err := clientContainer.alpaca.GetAccount()
		if err != nil {
			return err
		}
		adjSide := alpaca.Side(side)
		lastOrder, err := g.alpaca.PlaceOrder(alpaca.PlaceOrderRequest{
			AccountID:   account.ID,
			AssetKey:    &ticker,
			Qty:         decimal.NewFromFloat(float64(qty)),
			Side:        adjSide,
			Type:        "market",
			TimeInForce: "day",
		})
		if err == nil {
			fmt.Printf("Market order of | %d %s %s | completed.\n", qty, ticker, side)
			g.lastOrderID[ticker] = lastOrder.ID
		}
	}
	return nil
}

func (g *clientWrapper) LogSale(ticker, position string) {
	fmt.Printf("%s %s %.2f %.4f %.4f %.4f %.4f %.4f  %.4f %s\n", position, ticker, g.currPrice[ticker], g.ma[ticker].Value(), g.tema[ticker], g.momentum[ticker], g.momentum2[ticker], g.momentum3[ticker], g.mma[ticker].Value(), time.Now().Format("2006-01-02 15:04:05"))
}

