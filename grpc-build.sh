export GOPATH=~/go
export PATH=$PATH:$GOPATH/bin
protoc -I keystone/ keystone/keystone.proto --go_out=plugins=grpc:keystone
