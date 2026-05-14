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

	fmt.Println("Authorized:")
	r1 := &ap2v1.ListPaymentsRequest{Status: "Authorized"}
	resp1, err := client.ListPayments(ctx, r1)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range resp1.Payments {
		fmt.Println(p.TransactionId)
	}

	fmt.Println("\nDeclined:")
	r2 := &ap2v1.ListPaymentsRequest{Status: "Declined"}
	resp2, err := client.ListPayments(ctx, r2)
	if err != nil {
		log.Fatal(err)
	}
	for _, p := range resp2.Payments {
		fmt.Println(p.TransactionId)
	}
}
