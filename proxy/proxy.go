package proxy

import (
  "log"
  "net"
  "time"
  "errors"
  "strconv"
  "math/rand"
)
import "gopkg.in/fatih/pool.v2"
import "github.com/mikroio/tcp-forward-proxy/discovery"

type Config struct {
  MaxPoolSize int
  ConnectTimeout int
}

type targetState struct {
  pool pool.Pool
}

type Proxy struct {
  discovery *discovery.Discovery
  config Config
  states map[string]*targetState
  quit chan bool
  listener *net.TCPListener
}

func New(serviceDiscovery *discovery.Discovery, config Config) (proxy *Proxy) {
  proxy = new(Proxy)
  proxy.config = config
  proxy.discovery = serviceDiscovery
  proxy.states = make(map[string]*targetState)
  proxy.quit = make(chan bool)
  return proxy
}

func (proxy *Proxy) Service() string {
  return proxy.discovery.Service;
}

func (proxy *Proxy) Listen(port int) error {
  tcpAddr, err := net.ResolveTCPAddr("tcp", ":" + strconv.Itoa(port))
  if err != nil {
    return err
  }
  log.Print("start listening to", tcpAddr, port, err)
  proxy.listener, err = net.ListenTCP("tcp", tcpAddr)
  return err
}

func (proxy *Proxy) Accept() error {
  for {
    conn, err := proxy.listener.AcceptTCP()
    if err != nil {
      select {
        case <- proxy.quit:
          return nil
        default:
          log.Print("error while accepting", err)
          continue
      }
    }

    go proxy.handleConnection(conn)
  }
}

func (proxy *Proxy) Close() {
  proxy.quit <- false
  proxy.listener.Close()
}

func (proxy *Proxy) connectionFactory() (net.Conn, error) {
  items := proxy.discovery.Get()
  shuffle(items)

  for _, hostport := range items {
    state := proxy.ensureTargetState(hostport)
    conn, err := state.connect()
    if err == nil {
      return conn, nil
    }
  }

  return nil, errors.New("cannot find a endpoint that we can connect to")
}

func (proxy *Proxy) ensureTargetState(hostport string) (*targetState) {
  factory := func() (net.Conn, error) {
    return net.DialTimeout("tcp", hostport,
                           time.Second * time.Duration(proxy.config.ConnectTimeout))
  }

  if _, ok := proxy.states[hostport]; !ok {
    val := &targetState{}
    proxy.states[hostport] = val
    val.pool, _ = pool.NewChannelPool(0, proxy.config.MaxPoolSize, factory)
  }

  return proxy.states[hostport]
}

func (state *targetState) connect() (net.Conn, error) {
  return state.pool.Get()
}

func (proxy *Proxy) handleConnection(in net.Conn) error {
  defer in.Close()

  out, err := proxy.connectionFactory()
  if err != nil {
    log.Print("could no create outgoing connection", err)
    return err
  }
  defer out.Close()

  data := make([]byte, 1024 * 1024)

  for {
    n, err := in.Read(data)
    if err != nil {
      log.Printf("error reading data:", err)
      break
    }
    if n == 0 {
      break
    }

    _, err = out.Write(data[:n])
    if err != nil {
      log.Printf("error writing data:", err)
      break
    }
  }

  return nil
}

func shuffle(arr []string) {
  for i := len(arr) - 1; i > 0; i-- {
    j := rand.Intn(i)
    arr[i], arr[j] = arr[j], arr[i]
  }
}
