package main

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

var coinColors = []string{
	"\033[38;5;196m", // red
	"\033[38;5;46m",  // green
	"\033[38;5;33m",  // blue
	"\033[38;5;226m", // yellow
	"\033[38;5;208m", // orange
	"\033[38;5;201m", // magenta
	"\033[38;5;51m",  // cyan
	"\033[38;5;255m", // white
}

const colorReset = "\033[0m"
const colorDim = "\033[2m"
const colorBold = "\033[1m"

func printChart(hist *History, fiat string) {
	snaps := hist.All()
	if len(snaps) < 2 {
		return
	}

	// Collect all tickers (stable order)
	tickerSet := make(map[string]bool)
	for _, s := range snaps {
		for t := range s.Coins {
			tickerSet[t] = true
		}
	}
	tickers := make([]string, 0, len(tickerSet))
	for t := range tickerSet {
		tickers = append(tickers, t)
	}
	sort.Strings(tickers)

	colorMap := make(map[string]string)
	for i, t := range tickers {
		colorMap[t] = coinColors[i%len(coinColors)]
	}

	// Chart dimensions
	chartWidth := len(snaps)
	if chartWidth > 60 {
		chartWidth = 60
		snaps = snaps[len(snaps)-60:]
	}
	chartHeight := 15

	// Find global min/max
	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64
	for _, s := range snaps {
		for _, v := range s.Coins {
			if v < minVal {
				minVal = v
			}
			if v > maxVal {
				maxVal = v
			}
		}
	}

	// Add top padding and clamp bottom to 0
	valRange := maxVal - minVal
	if valRange == 0 {
		valRange = 1
	}
	maxVal += valRange * 0.05
	if minVal > 0 {
		minVal = 0
	}
	valRange = maxVal - minVal

	// Build grid: grid[row][col] = character to print
	// row 0 = top (maxVal), row chartHeight-1 = bottom (minVal)
	type cell struct {
		char  rune
		color string
	}
	grid := make([][]cell, chartHeight)
	for r := range grid {
		grid[r] = make([]cell, chartWidth)
		for c := range grid[r] {
			grid[r][c] = cell{char: ' '}
		}
	}

	// Plot each coin using box-drawing characters with rounded corners
	for _, ticker := range tickers {
		color := colorMap[ticker]
		for col, s := range snaps {
			v, ok := s.Coins[ticker]
			if !ok {
				continue
			}
			row := int(float64(chartHeight-1) * (1 - (v-minVal)/valRange))
			if row < 0 {
				row = 0
			}
			if row >= chartHeight {
				row = chartHeight - 1
			}
			ch := '─'
			// Connect with rounded corners if previous point was at a different row
			if col > 0 {
				prevV, ok := snaps[col-1].Coins[ticker]
				if ok {
					prevRow := int(float64(chartHeight-1) * (1 - (prevV-minVal)/valRange))
					if prevRow < 0 {
						prevRow = 0
					}
					if prevRow >= chartHeight {
						prevRow = chartHeight - 1
					}
					if prevRow > row {
						// Going UP: line ascends from prevRow to row
						// ╭─  (row: arrival, corner DOWN+RIGHT)
						// │   (middle: vertical pass-through)
						// ╯   (prevRow: entry corner LEFT+UP)
						if grid[prevRow][col].char == ' ' {
							grid[prevRow][col] = cell{char: '╯', color: color}
						}
						for r := row + 1; r < prevRow; r++ {
							if grid[r][col].char == ' ' {
								grid[r][col] = cell{char: '│', color: color}
							}
						}
						ch = '╭'
					} else if prevRow < row {
						// Going DOWN: line descends from prevRow to row
						// ╮   (prevRow: entry corner LEFT+DOWN)
						// │   (middle: vertical pass-through)
						// ╰─  (row: arrival, corner UP+RIGHT)
						if grid[prevRow][col].char == ' ' {
							grid[prevRow][col] = cell{char: '╮', color: color}
						}
						for r := prevRow + 1; r < row; r++ {
							if grid[r][col].char == ' ' {
								grid[r][col] = cell{char: '│', color: color}
							}
						}
						ch = '╰'
					}
				}
			}
			// Mark mining coin with a bold dot
			if s.Mining == ticker {
				ch = '●'
			}
			grid[row][col] = cell{char: ch, color: color}
		}
	}

	// Mark switch events with a vertical dashed line
	for col, s := range snaps {
		if s.Switched {
			for r := 0; r < chartHeight; r++ {
				if grid[r][col].char == ' ' {
					grid[r][col] = cell{char: '┊', color: colorDim}
				}
			}
		}
	}

	// Render
	currency := strings.ToUpper(fiat)
	fmt.Printf("\n  %sProfitability Chart (%s/day)%s\n", colorBold, currency, colorReset)

	for r := 0; r < chartHeight; r++ {
		// Y-axis label (5 positions: top, middle, bottom)
		val := maxVal - float64(r)/float64(chartHeight-1)*valRange
		if r == 0 || r == chartHeight-1 || r == chartHeight/2 {
			fmt.Printf("  %10.6f │", val)
		} else {
			fmt.Printf("             │")
		}
		for c := 0; c < chartWidth; c++ {
			cl := grid[r][c]
			if cl.color != "" {
				fmt.Printf("%s%c%s", cl.color, cl.char, colorReset)
			} else {
				fmt.Printf("%c", cl.char)
			}
		}
		fmt.Println()
	}

	// X-axis
	fmt.Printf("             └")
	fmt.Print(strings.Repeat("─", chartWidth))
	fmt.Println()

	// Time labels
	if len(snaps) > 0 {
		first := snaps[0].Time.Format("15:04")
		last := snaps[len(snaps)-1].Time.Format("15:04")
		pad := chartWidth - len(first) - len(last)
		if pad < 1 {
			pad = 1
		}
		fmt.Printf("              %s%s%s\n", first, strings.Repeat(" ", pad), last)
	}

	// Legend
	fmt.Print("  ")
	for _, t := range tickers {
		fmt.Printf(" %s●%s %s", colorMap[t], colorReset, t)
	}
	fmt.Printf("   %s┊%s = switch\n\n", colorDim, colorReset)
}
