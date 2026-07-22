package main

import (
	"fmt"
	"math/rand"
	"time"
	"math"

)

func handleRequest() {
	ms := rand.Intn(100) + 10
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func handleRequestB(){
	ms := rand.Intn(100) + 10
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func operation(a , b float64){
	result := math.Pow(a,b)  
	fmt.Println(result)
	time.Sleep(time.Duration(10) * time.Millisecond)
}

func main() {
	fmt.Print("This is a test")
	for {
		handleRequest()
		handleRequestB()
		operation(1000,2000)
		fmt.Println("its being handle")
	}
}
