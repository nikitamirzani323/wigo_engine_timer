package main

import (
	"fmt"
	"time"
)

func main() {
	time_game := 30
	time_compile := 20
	flag := true

	for flag {
		flag = loop_count(time_game, time_compile)
	}

}
func loop_count(sec, compile int) bool {
	flag := false
	fmt.Println("")
	for sec >= 0 {
		fmt.Printf("%.2d\r", sec%60)
		time.Sleep(1 * time.Second)
		sec--
	}
	flag = loop_compile(compile)
	return flag
}
func loop_compile(sec int) bool {
	flag_compile := false
	fmt.Println("")
	for sec >= 0 {
		fmt.Printf("COMPILE %.2d\r", sec%60)
		time.Sleep(1 * time.Second)
		sec--
	}
	flag_compile = true
	return flag_compile
}
