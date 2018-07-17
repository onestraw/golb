package main

import (
	"fmt"
	"strings"
	"time"

	clientv3 "github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/etcdserver/api/v3rpc/rpctypes"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

var (
	etcd_client *clientv3.Client
	serviceKey  string
	stopSignal  = make(chan bool, 1)
)

// Register add serverAddr to poolName in etcd endpoints
func Register(prefix, poolName, serverAddr, endpoints string, interval time.Duration, TTL int64) error {
	serviceValue := serverAddr
	serviceKey = fmt.Sprintf("%s/virtualserver/%s/pool/%s/address", prefix, poolName, serverAddr)

	client, err := clientv3.New(clientv3.Config{
		Endpoints: strings.Split(endpoints, ","),
	})
	if err != nil {
		return fmt.Errorf("create etcd client failed: %v", err)
	}
	etcd_client = client

	// interval should be less than TTL
	// it's heartbeat, update the lease every interval before the lease expired
	go func() {
		ticker := time.NewTicker(interval)
		for {
			resp, _ := client.Grant(context.TODO(), int64(TTL))
			if _, err := client.Get(context.Background(), serviceKey); err != nil {
				if err == rpctypes.ErrKeyNotFound {
					if _, err := client.Put(context.TODO(), serviceKey, serviceValue, clientv3.WithLease(resp.ID)); err != nil {
						log.Errorf("set service %q with ttl to etcd failed: %s", poolName, err.Error())
					}
				} else {
					log.Errorf("service %q connect to etcd failed: %s", poolName, err.Error())
				}
			} else {
				if _, err := client.Put(context.Background(), serviceKey, serviceValue, clientv3.WithLease(resp.ID)); err != nil {
					log.Errorf("refresh service %q with ttl to etcd failed: %s", poolName, err.Error())
				}
			}

			select {
			case <-stopSignal:
				ticker.Stop()
				return
			case <-ticker.C:
			}
		}
	}()
	return nil
}

// UnRegister remove server from pool
func UnRegister() error {
	stopSignal <- true
	stopSignal = make(chan bool, 1) // just a hack to avoid multi UnRegister deadlock
	if _, err := etcd_client.Delete(context.Background(), serviceKey); err != nil {
		log.Errorf("unregister %q failed: %s.", serviceKey, err.Error())
		return err
	}
	log.Infof("unregister %v done.", serviceKey)
	return nil
}
