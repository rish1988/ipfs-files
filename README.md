# Ipfs-files

Provides the following features:

* Start up an in IPFS node
* Configure IPFS node with custom options in `config.IpfsOptions`
* Provide access to `CoreAPI` and `FileAPI` over the exported `IpfsAPI`

# Usage

* Import the package `github.com/rish1988/ipfs-files`

* Starting an IPFS node is simple as using the below code

  

```go
package main

import (
	"context"
	"github.com/rish1988/ipfs-files"
)

func main() {
	
  // Get a context to work with
  ctx, cancel := context.WithCancel(context.context.Backround())
	
  // When the context is destroyed, the IPFS node is also destroyed
  defer cancel()
	
  // Set some options
  opts := config.IpfsOptions {
    NodeType: config.Default,
    BootstrapNodes: []string {
      "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
			"/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    },
    EnableExperimental: true,
  }

  // Get access to Ipfs CoreAPI and FileAPI
  ipfsApi, err := ipfs_files.IpfsNode(ctx, opts)
}

```
