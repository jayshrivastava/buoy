# buoy

A simplified implementation of a distributed key value store which uses the Raft concensus algorithm. 

This project launches a cluster of raft nodes which actively log what is currently taking place in the cluster. A client interface is also provided for sending data to the cluster.

## Resources
- A practical [guide](https://eli.thegreenplace.net/2020/implementing-raft-part-0-introduction/) to implementing Raft.  
- The Raft [paper](https://raft.github.io/raft.pdf)

## Development  
Download Deps
`go mod download`  

Generate Protocol Buffer for node <-> node RPCs
`protoc -I node/ node/raft.proto --go_out=plugins=grpc:node`

Generate Protocol Buffer for client <-> cluster RPCs
`protoc -I api/ api/api.proto --go_out=plugins=grpc:api`

Formatting  
`go fmt ./...`
