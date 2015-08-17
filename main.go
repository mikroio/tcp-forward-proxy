package main

import (
  "flag"
)

import "github.com/mikroio/tcp-forward-proxy/discovery"
import "github.com/mikroio/tcp-forward-proxy/proxy"

var listenPort *int = flag.Int("l", 8080, "listen port")
var service *string = flag.String("s", "", "service name")

func main() {
  flag.Parse()

  serviceDiscovery := discovery.New(*service, "eu-west-1", "uio-jrydberg-default")
  serviceDiscovery.Start()

  config := proxy.Config{
    MaxPoolSize: 30,
    ConnectTimeout: 10,
  }

  forwardProxy := proxy.New(serviceDiscovery, config)
  err := forwardProxy.Listen(*listenPort)
  if err != nil {
    panic(err)
  }

  forwardProxy.Accept()
}
