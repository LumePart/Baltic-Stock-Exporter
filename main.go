package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"io"

	"github.com/xuri/excelize/v2"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


var prom_reg *prometheus.Registry


func downloadFile(url string, localFilePath string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: %s", response.Status)
	}

	file, err := os.Create(localFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func readAllStocks() [][]string { // Reads an Excel file named and returns a 2D string slice representing the "Shares" sheet data.
	stock_url := "https://nasdaqbaltic.com/statistics/en/shares?download=1"
	file_path := "/tmp/shares.xlsx"
	downloadFile(stock_url,file_path)

	f, err := excelize.OpenFile(file_path)
    if err != nil {
        fmt.Println(err)
    }
    defer func() {
        if err := f.Close(); err != nil {
            fmt.Println(err)
        }
    }()

    rows, err := f.GetRows("Shares")
    if err != nil {
        fmt.Println(err)
    }
    return rows
}

func getStockLabels(row []string) []string { // Extracts specific fields from a row of stock data to create and return a list of metric labels.
	tags := make([]string, 0, 6)
	for _, val := range []int{0, 1, 4, 18, 19} {
		tags = append(tags, strings.ToLower(row[val]))
	}
	tags = append(tags, strings.Split(row[5], "/")[1])
	return tags
}

func main() {
	prom_reg = prometheus.NewRegistry()
	stocks := regStocks()

	prometheus.DefaultRegisterer = prom_reg
	prometheus.DefaultGatherer = prom_reg


	collect := &CustomCollector{
		stockPriceMetric: stocks.stockPriceMetric,
		stockTradeMetric: stocks.stockTradeMetric,
		stockVolumeMetric: stocks.stockVolumeMetric,
		stockTurnoverMetric: stocks.stockTurnoverMetric,
	}
	register(collect)

	http.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
	http.ListenAndServe(":33171", nil)
}
