# buoy

A simplified implementation of a distributed key value store which uses the Raft concensus algorithm. 

## Resources
- A practical [guide](https://eli.thegreenplace.net/2020/implementing-raft-part-0-introduction/) to implementing Raft.  
- The Raft [paper](https://raft.github.io/raft.pdf)

## Development  
Download Deps
`go mod download`  

Generate Protocol Bufffer    
`protoc -I rpc/ rpc/raft.proto --go_out=plugins=grpc:rpc`

Formatting  
`go fmt ./...`
