package main

// Realistic-looking target binary for the eBPF profiler: a fake order
// processing pipeline with a layered call graph (handler -> auth ->
// business logic -> data layer -> payment -> response), instead of
// three flat functions. Gives the graph something that actually looks
// like a service instead of a toy chain.
//
// Call graph shape (per request):
//
//   main
//    └─ handleOrderRequest
//        ├─ authenticateUser
//        │   └─ loadUserSession
//        ├─ validateOrder
//        │   └─ checkInventory
//        ├─ calculatePricing
//        │   └─ applyDiscounts
//        ├─ processPayment          <-- regression injected here
//        │   └─ chargeCard
//        ├─ persistOrder
//        │   └─ writeOrderToDB
//        ├─ sendConfirmationEmail
//        └─ logAuditTrail
//
// BUILD (disable inlining, or uprobes on small funcs may never fire):
//   go build -gcflags="all=-l" -o target_service target_service.go
//
// RUN:
//   ./target_service
//
// Then, in another terminal:
//   sudo ./profiler ./target_service
//
// The first ~25 requests run at normal latency and establish a baseline.
// After that, chargeCard (deep in the payment path) starts taking 100x
// longer, simulating a slow payment gateway -- should trigger Regression
// on main.chargeCard and cascade visually up through processPayment.

import (
	"fmt"
	"math/rand"
	"time"
)

const (
	totalRequests = 60
	spikeStartsAt = 25

	// baseline latencies per stage, tuned to feel like a real request
	authLatency      = 3 * time.Millisecond
	sessionLatency   = 2 * time.Millisecond
	validateLatency  = 2 * time.Millisecond
	inventoryLatency = 4 * time.Millisecond
	pricingLatency   = 1 * time.Millisecond
	discountLatency  = 1 * time.Millisecond
	paymentLatency   = 6 * time.Millisecond
	dbLatency        = 5 * time.Millisecond
	emailLatency     = 2 * time.Millisecond
	auditLatency     = 1 * time.Millisecond

	// the regression: chargeCard goes from ~6ms to 600ms (100x)
	chargeCardBaseline = 6 * time.Millisecond
	chargeCardSpike    = 600 * time.Millisecond
)

//go:noinline
func loadUserSession(orderID int) {
	time.Sleep(sessionLatency)
}

//go:noinline
func authenticateUser(orderID int) {
	time.Sleep(authLatency)
	loadUserSession(orderID)
}

//go:noinline
func checkInventory(orderID int) {
	time.Sleep(inventoryLatency)
}

//go:noinline
func validateOrder(orderID int) {
	time.Sleep(validateLatency)
	checkInventory(orderID)
}

//go:noinline
func applyDiscounts(orderID int) {
	time.Sleep(discountLatency)
}

//go:noinline
func calculatePricing(orderID int) {
	time.Sleep(pricingLatency)
	applyDiscounts(orderID)
}

//go:noinline
func chargeCard(orderID int, spiking bool) {
	if spiking {
		time.Sleep(chargeCardSpike)
	} else {
		time.Sleep(chargeCardBaseline)
	}
}

//go:noinline
func processPayment(orderID int, spiking bool) {
	time.Sleep(paymentLatency)
	chargeCard(orderID, spiking)
}

//go:noinline
func writeOrderToDB(orderID int) {
	time.Sleep(dbLatency)
}

//go:noinline
func persistOrder(orderID int) {
	writeOrderToDB(orderID)
}

//go:noinline
func sendConfirmationEmail(orderID int) {
	time.Sleep(emailLatency)
}

//go:noinline
func logAuditTrail(orderID int) {
	time.Sleep(auditLatency)
}

//go:noinline
func handleOrderRequest(orderID int, spiking bool) {
	authenticateUser(orderID)
	validateOrder(orderID)
	calculatePricing(orderID)
	processPayment(orderID, spiking)
	persistOrder(orderID)
	sendConfirmationEmail(orderID)
	logAuditTrail(orderID)
}

func main() {
	fmt.Printf("target_service starting: %d sequential requests, payment regression after request #%d\n",
		totalRequests, spikeStartsAt)

	for i := 0; i < totalRequests; i++ {
		spiking := i >= spikeStartsAt
		if i == spikeStartsAt {
			fmt.Println("!! REGRESSION INJECTED -- chargeCard just went 100x slower !!")
		}
		handleOrderRequest(i, spiking)
		// small jitter between requests so events are easy to watch land one at a time
		time.Sleep(time.Duration(40+rand.Intn(20)) * time.Millisecond)
	}

	fmt.Println("target_service done")
}
