// Service discovery! YAY!

package discovery

import (
  "log"
  "net"
  "time"
  "sync"

  "github.com/AdRoll/goamz/aws"
  "github.com/AdRoll/goamz/dynamodb"
)

var tableDescription = dynamodb.TableDescriptionT{
  AttributeDefinitions: []dynamodb.AttributeDefinitionT{
    dynamodb.AttributeDefinitionT{"Service", "S"},
    dynamodb.AttributeDefinitionT{"TaskID", "S"},
    },
  KeySchema: []dynamodb.KeySchemaT{
    dynamodb.KeySchemaT{"Service", "HASH"},
    dynamodb.KeySchemaT{"TaskID", "RANGE"},
    },
}

type Discovery struct {
  sync.Mutex
  table *dynamodb.Table
  Service string
  items []string
  quit chan bool
}

func New(service string, region string, tableName string) *Discovery {
  pk, err := tableDescription.BuildPrimaryKey()
  if err != nil {
    log.Fatal(err)
  }

  auth, err := aws.GetAuth("", "", "", time.Now())
  dbServer := dynamodb.New(auth, aws.GetRegion(region))
  table := dbServer.NewTable(tableName, pk)
  return &Discovery{table: table, Service: service}
}

func (d *Discovery) Start() {
  go func() {
    t := time.NewTicker(time.Second * 15)
    for {
      d.query()
      select {
        case <- t.C:
        case <- d.quit:
          t.Stop()
          break
      }
    }
  }()
}

func (d *Discovery) Close() {
  close(d.quit)
}

func (d *Discovery) query() {
  q := dynamodb.NewQuery(d.table)
  q.AddKeyConditions([]dynamodb.AttributeComparison{
    *dynamodb.NewEqualStringAttributeComparison("Service", d.Service),
  })

  result, _, err := d.table.QueryTable(q)

  if err != nil {
    log.Println("cannot query right now:", err)
    return
  }

  items := make([]string, 0)

  for _, item := range result {
    hostAddr := item["HostAddr"].Value
    hostPort := item["HostPort"].Value
    items = append(items, net.JoinHostPort(hostAddr, hostPort))
  }

  log.Printf("we got %d items", len(items))

  d.Lock()
  defer d.Unlock()
  d.items = items
}

// Return all current discovered endpoints for the service.
func (d *Discovery) Get() []string {
  d.Lock()
  defer d.Unlock()

  items := make([]string, len(d.items))
  for i, item := range d.items {
    items[i] = item
  }

  return items
}
