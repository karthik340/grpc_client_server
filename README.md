# grpc_client_server
example that shows communication between grpc client and server in go
compile proto using protoc -I. --go-grpc_out=require_unimplemented_servers=false:../ --go_out=../ product_info.proto <br/>
build server and client using go build -i -v -o bin/server <br/>
