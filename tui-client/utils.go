package main

import "fmt"

func msToMMSS(ms int) string {
	total := ms / 1000
	return fmt.Sprintf("%d:%02d", total/60, total%60)
}