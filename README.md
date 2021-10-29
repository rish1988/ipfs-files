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
	ipfsfiles "github.com/rish1988/ipfs-files"
)

func main() {
	
  // Get a context to work with
  ctx, cancel := context.WithCancel(context.context.Backround())
	
  // When the context is destroyed, the IPFS node is also destroyed
  defer cancel()
	
  // Set some options
  opts := ipfsfiles.IpfsOptions {
    // Default or ephemeral node type
    NodeType: ipfsfiles.Default,
    
    // Nodes to connect to
    BootstrapNodes: []string {
      "/dnsaddr/bootstrap.libp2p.io/p2p/QmNnooDu7bfjPFoTZYxMNLWUQJyrVwtbZg5gBMjTezGAJN",
      "/dnsaddr/bootstrap.libp2p.io/p2p/QmQCU2EcMqAqQPR2i9bChDtGNJchTbq5TbXJJ16u19uLTa",
      "/dnsaddr/bootstrap.libp2p.io/p2p/QmbLHAnMoJPWSCR5Zhtx6BHJX9KiKNN6tpvbUcqanj75Nb",
      "/dnsaddr/bootstrap.libp2p.io/p2p/QmcZf59bWwK5XFi76CZX8cbJ4BhTzzA3gU1ZjYZcYW3dwt",
    },
    
    // Enable experimental IPFS features
    EnableExperimental: true,
    
    // Use these options to spin up private IPFS network
    // All nodes in the network share the swarm key and have 
    // access to all files
    SwarmOptions : &ipfsfiles.SwarmOptions {
    	Private: true,
	
	// Optionally set the key or let it be generated for you
	Key : "f509aeebecf2547a51614e4705b2dafa44f3d04c6b23aebde1091b3002acb7ae"
    }
  }

  // Get access to Ipfs CoreAPI and FileAPI
  ipfsApi, err := ipfsfiles.IpfsNode(ctx, opts)
}

```
