package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"

	pb "productinfo/service/ecommerce"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	wrapper "google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	port           = ":50051"
	orderBatchSize = 3
)

//sudo go build -i -v -o bin/server
// protoc -I. --go-grpc_out=require_unimplemented_servers=false:../ --go_out=../ product_info.proto

type server struct {
	productMap map[string]*pb.Product
}

var orderMap = make(map[string]*pb.Order)

func (s *server) AddProduct(ctx context.Context, in *pb.Product) (*pb.ProductID, error) {
	out, err := uuid.NewV4()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error while generating Product ID", err)
	}
	in.Id = out.String()
	if s.productMap == nil {
		s.productMap = make(map[string]*pb.Product)
	}
	s.productMap[in.Id] = in
	return &pb.ProductID{Value: in.Id}, status.New(codes.OK, "").Err()
}

func (s *server) GetProduct(ctx context.Context, in *pb.ProductID) (*pb.Product, error) {
	value, exists := s.productMap[in.Value]
	if exists {
		return value, status.New(codes.OK, "").Err()
	}
	return nil, status.Errorf(codes.NotFound, "product does not exist", in.Value)
}

//Simple Rpc
func (s *server) GetOrder(ctx context.Context, orderId *wrapper.StringValue) (*pb.Order, error) {

	ord, exists := orderMap[orderId.Value]
	if exists {
		return ord, status.New(codes.OK, "").Err()
	}
	return nil, status.Errorf(codes.NotFound, "order does not exist : ", orderId)
}

//Server Streaming RPC
func (s *server) SearchOrders(searchQuery *wrapper.StringValue, stream pb.OrderManagement_SearchOrdersServer) error {
	fmt.Println("hello")
	for key, order := range orderMap {
		log.Println(key, order)
		for _, itemStr := range order.Items {
			log.Println(itemStr)
			if strings.Contains(itemStr, searchQuery.Value) {
				err := stream.Send(order)
				if err != nil {
					return fmt.Errorf("error sending message to stream: %v", err)
				}
				log.Println("matching order found : " + key)
				break
			}
		}
	}
	return nil
}

//Client Streaming RPC
func (s *server) UpdateOrders(stream pb.OrderManagement_UpdateOrdersServer) error {

	ordersStr := "Updated Order IDs : "
	for {
		order, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&wrapper.StringValue{Value: "orders processed " + ordersStr})
		}
		orderMap[order.Id] = order
		fmt.Println("Order ID : ", order.Id, ":Updated")
		ordersStr += order.Id + ", "
	}
}

// Bi-directional Streaming RPC
func (s *server) ProcessOrders(stream pb.OrderManagement_ProcessOrdersServer) error {

	batchMarker := 1
	var combinedShipmentMap = make(map[string]*pb.CombinedShipment)
	for {
		orderId, err := stream.Recv()
		log.Printf("Reading Proc order : %s", orderId)
		if err == io.EOF {
			// Client has sent all the messages
			// Send remaining shipments
			log.Printf("EOF : %s", orderId)
			for _, shipment := range combinedShipmentMap {
				if err := stream.Send(shipment); err != nil {
					return err
				}
			}
			return nil
		}
		if err != nil {
			log.Println(err)
			return err
		}

		destination := orderMap[orderId.GetValue()].Destination
		shipment, found := combinedShipmentMap[destination]
		if found {
			ord := orderMap[orderId.GetValue()]
			shipment.OrderList = append(shipment.OrderList, ord)
			combinedShipmentMap[destination] = shipment
		} else {
			shipment := pb.CombinedShipment{}
			comShip := pb.CombinedShipment{Id: "cmb - " + (orderMap[orderId.GetValue()].Destination), Status: "Processed!"}
			ord := orderMap[orderId.GetValue()]
			comShip.OrderList = append(shipment.OrderList, ord)
			combinedShipmentMap[destination] = &comShip
			log.Print(len(comShip.OrderList), comShip.GetId())
		}

		if batchMarker == orderBatchSize {
			for _, comb := range combinedShipmentMap {
				log.Printf("Shipping : %v -> %v", comb.Id, len(comb.OrderList))
				if err := stream.Send(comb); err != nil {
					return err
				}
			}
			batchMarker = 0
			combinedShipmentMap = make(map[string]*pb.CombinedShipment)
		} else {
			batchMarker++
		}
	}
}

func main() {
	initSampleData()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen : %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterOrderManagementServer(s, &server{})

	log.Printf("Starting grpc listener on port " + port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve : %v", err)
	} else {
		fmt.Println("hello1")
	}
}

func initSampleData() {
	orderMap["102"] = &pb.Order{Id: "102", Items: []string{"Google Pixel 3A", "Mac Book Pro"}, Destination: "Mountain View, CA", Price: 1800.00}
	orderMap["103"] = &pb.Order{Id: "103", Items: []string{"Apple Watch S4"}, Destination: "San Jose, CA", Price: 400.00}
	orderMap["104"] = &pb.Order{Id: "104", Items: []string{"Google Home Mini", "Google Nest Hub"}, Destination: "Mountain View, CA", Price: 400.00}
	orderMap["105"] = &pb.Order{Id: "105", Items: []string{"Amazon Echo"}, Destination: "San Jose, CA", Price: 30.00}
	orderMap["106"] = &pb.Order{Id: "106", Items: []string{"Amazon Echo", "Apple iPhone XS"}, Destination: "Mountain View, CA", Price: 300.00}
}
