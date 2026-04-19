package main

import (
	"context"
	"fmt"
	"log"
	"time"

	ap2v1 "github.com/Gorkyichocolate/ap2-generated/ap2/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Подключаемся к Payment Service gRPC
	conn, err := grpc.DialContext(
		context.Background(),
		"localhost:8088",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := ap2v1.NewPaymentServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test 1: Создаём несколько платежей с разными статусами
	fmt.Println("=== Test 1: Creating payments ===")
	paymentRequests := []*ap2v1.PaymentRequest{
		{OrderId: "order-1", Amount: 1000},  // Authorized
		{OrderId: "order-2", Amount: 60000}, // Declined
		{OrderId: "order-3", Amount: 2000},  // Authorized
	}

	for _, req := range paymentRequests {
		resp, err := client.ProcessPayment(ctx, req)
		if err != nil {
			log.Printf("Failed to process payment: %v", err)
			continue
		}
		fmt.Printf("Order %s: Status=%s, TransactionID=%s\n", req.OrderId, resp.Status, resp.TransactionId)
	}

	// Test 2: Список платежей со статусом "Authorized"
	fmt.Println("\n=== Test 2: ListPayments - Authorized ===")
	listReq := &ap2v1.ListPaymentsRequest{Status: "Authorized"}
	listResp, err := client.ListPayments(ctx, listReq)
	if err != nil {
		log.Fatalf("Failed to list payments: %v", err)
	}

	fmt.Printf("Found %d authorized payments:\n", len(listResp.Payments))
	for i, p := range listResp.Payments {
		fmt.Printf("  [%d] Status=%s, TransactionID=%s\n", i+1, p.Status, p.TransactionId)
	}

	// Test 3: Список платежей со статусом "Declined"
	fmt.Println("\n=== Test 3: ListPayments - Declined ===")
	listReq = &ap2v1.ListPaymentsRequest{Status: "Declined"}
	listResp, err = client.ListPayments(ctx, listReq)
	if err != nil {
		log.Fatalf("Failed to list payments: %v", err)
	}

	fmt.Printf("Found %d declined payments:\n", len(listResp.Payments))
	for i, p := range listResp.Payments {
		fmt.Printf("  [%d] Status=%s, TransactionID=%s\n", i+1, p.Status, p.TransactionId)
	}

	fmt.Println("\n✓ All tests completed!")
}
