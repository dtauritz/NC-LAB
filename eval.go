package main

import (
	// "github.com/Knetic/govaluate"
	"fmt"
	"bufio"
	"os"
	"strconv"
)

type test struct {
	attr 	map[string]float64
}

func testConstructor() test {
	var result test
	result.attr = make(map[string]float64)
	return result
}

func testCopy(in test) test {
	result := in

	result.attr = make(map[string]float64)

	for key, val := range in.attr {
		result.attr[key] = val
	}

	return result
}

func testFile(filePath string) test {
	result := testConstructor()

	file, err := os.Open(filePath)
	if err != nil {
		panic(0)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		if err != nil {
			panic(0)
		}
		key := scanner.Text()
		scanner.Scan()
		val := scanner.Text()
		fmt.Printf("val %v\n", val)
		tmp, err := strconv.ParseFloat(val, 64)
		if err != nil {
			panic(1)
		}
		result.attr[key] = tmp
	}

	return result
}

func main() {
	x := testFile("metal.txt")
	y := testCopy(x)

	y.attr["two"] = 4.01
	y.attr["three"] = 5.98676


	fmt.Printf("x: %v\n", x.attr)
	fmt.Printf("y: %v\n", y.attr)
}