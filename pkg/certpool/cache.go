package certpool

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"sync"
)

type Cache struct {
	RootCAs *x509.CertPool
}

var cacheOnce sync.Once
var cacheInstance *Cache

func GetCache() *Cache {
	cacheOnce.Do(func() {
		rootCAs, _ := x509.SystemCertPool()
		if rootCAs == nil {
			rootCAs = x509.NewCertPool()
		}
		cacheInstance = &Cache{
			RootCAs: rootCAs,
		}
	})
	return cacheInstance
}

func (c *Cache) AddCerts(certfile string) error {
	// Read in the cert file
	certs, err := ioutil.ReadFile(certfile)
	if err != nil {
		return fmt.Errorf("Failed to append %q to RootCAs: %v", certfile, err)
	}

	// Append our cert to the system pool
	if ok := c.RootCAs.AppendCertsFromPEM(certs); !ok {
		log.Println("No certs appended, using system certs only")
	}
	return nil
}