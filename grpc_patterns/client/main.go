package main

import (
	"context"
	"io"
	"log"
	pb "productinfo/client/ecommerce"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	wrapper "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	address = "localhost:50051"
)

//sudo go build -i -v -o bin/client
// protoc -I. --go-grpc_out=require_unimplemented_servers=false:../ --go_out=../ product_info.proto

func main() {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect : %v", err)
	}
	defer conn.Close()
	client := pb.NewOrderManagementClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	// //Simple Rpc
	// retrieveOrder, err := client.GetOrder(ctx, &wrapper.StringValue{Value: "106"})
	// if err != nil {
	// 	log.Print("did not get Response", err)
	// } else {
	// 	log.Print("Get Order Response -> :", retrieveOrder)
	// }

	// //Server Streaming RPC
	// searchStream, err := client.SearchOrders(ctx, &wrapper.StringValue{Value: "Google"})
	// if err != nil {
	// 	log.Print("did not get Response", err)
	// } else {
	// 	for {
	// 		searchOrder, err := searchStream.Recv()
	// 		if err == io.EOF {
	// 			break
	// 		}
	// 		log.Print("Search Result :", searchOrder)
	// 	}
	// }

	// // // =========================================
	// // Update Orders : Client streaming scenario
	// updOrder1 := pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Google Pixel Book"}, Destination: "Mountain View, CA", Price: 1100.00}
	// updOrder2 := pb.Order{Id: "103", Items: []string{"Apple Watch S4", "Mac Book Pro", "iPad Pro"}, Destination: "San Jose, CA", Price: 2800.00}
	// updOrder3 := pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub", "iPad Mini"}, Destination: "Mountain View, CA", Price: 2200.00}

	// //Client Streaming RPC
	// updateStream, err := client.UpdateOrders(ctx)
	// if err != nil {
	// 	log.Fatalf("%v.UpdateOrders(_) = ,%v", client, err)
	// }

	// //Updating order 1
	// if err := updateStream.Send(&updOrder1); err != nil {
	// 	log.Fatalf("%v.Send(%v) = %v", updateStream, &updOrder1, err)
	// }

	// //Updating order 2
	// if err := updateStream.Send(&updOrder2); err != nil {
	// 	log.Fatalf("%v.Send(%v) = %v", updateStream, &updOrder2, err)
	// }

	// //Updating order 3
	// if err := updateStream.Send(&updOrder3); err != nil {
	// 	log.Fatalf("%v.Send(%v) = %v", updateStream, &updOrder3, err)
	// }

	// updateRes, err := updateStream.CloseAndRecv()
	// if err != nil {
	// 	log.Fatalf("%v.CloseAndRecv() got error %v, want %v", updateStream, err, nil)
	// }
	// log.Printf("Update Orders Res : %s", updateRes)

	// =========================================
	// Process Order : Bi-di streaming scenario
	streamProcOrder, err := client.ProcessOrders(ctx)
	if err != nil {
		log.Fatalf("%v.ProcessOrders(_) = _, %v", client, err)
	}

	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "102"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "102", err)
	}

	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "103"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "103", err)
	}

	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "104"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "104", err)
	}

	channel := make(chan struct{})
	go asncClientBidirectionalRPC(streamProcOrder, channel)
	time.Sleep(time.Millisecond * 1000)

	if err := streamProcOrder.Send(&wrapper.StringValue{Value: "101"}); err != nil {
		log.Fatalf("%v.Send(%v) = %v", client, "101", err)
	}
	if err := streamProcOrder.CloseSend(); err != nil {
		log.Fatal(err)
	}
	channel <- struct{}{}
}

func asncClientBidirectionalRPC(streamProcOrder pb.OrderManagement_ProcessOrdersClient, c chan struct{}) {
	for {
		combinedShipment, errProcOrder := streamProcOrder.Recv()
		if errProcOrder == io.EOF {
			break
		}
		log.Printf("Combined shipment : %v", combinedShipment.OrderList)
	}
	<-c
}
