package ipfs

import (
	"context"
	"fmt"
	config "github.com/ipfs/go-ipfs-config"
	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreapi"
	"github.com/ipfs/go-ipfs/core/node/libp2p"
	"github.com/ipfs/go-ipfs/plugin/loader" // This package is needed so that all the preloaded plugins are loaded automatically
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	icore "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
	ipfsconfig "github.com/rish1988/ipfs-files/config"
	"github.com/rish1988/ipfs-files/db"
	"github.com/rish1988/ipfs-files/log"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type IpfsApi interface {
	FilesApi() *FileApi
	CoreApi() *icore.CoreAPI
}

type Ipfs struct {
	node    *core.IpfsNode
	CoreAPI *icore.CoreAPI
	*FileApi
	*ipfsconfig.IpfsOptions
}

// StartIpfsNode starts a new IPFS ipfs (default or ephemeral), connecting to it configured
// peers, and wraps an API to add or get ipfs and directories.
func StartIpfsNode(ctx context.Context, options ipfsconfig.IpfsOptions) (IpfsApi, error) {
	var (
		node *core.IpfsNode
		err  error
		i    *Ipfs
		api  icore.CoreAPI
	)

	i = &Ipfs{
		IpfsOptions: &options,
	}

	/// --- Part I: Getting a IPFS ipfs running

	log.Info("-- Getting an IPFS ipfs running -- ")

	if len(options.NodeType) == 0 {
		options.NodeType = ipfsconfig.Ephemeral
	}

	if i.NodeType == ipfsconfig.Default {
		// Spawn a ipfs using the default path (~/.ipfs), assuming that a repo exists there already
		log.Info("Spawning ipfs on default repo")
		node, err = i.spawnDefault(ctx)
		if err != nil {
			log.Errorf("Failed to spawnDefault ipfs: %s", err)
			return nil, err
		}
	} else if i.NodeType == ipfsconfig.Ephemeral {
		log.Info("Spawning ipfs on a temporary repo")
		if node, err = i.spawnEphemeral(ctx); err != nil {
			log.Errorf("Failed to spawn ephemeral ipfs: %s", err)
			return nil, err
		}
	}

	if api, err = coreapi.NewCoreAPI(node); err != nil {
		log.Errorf("Failed to attach Core API to Node", err)
	} else {
		i.node = node
		i.CoreAPI = &api
	}

	log.Info("IPFS ipfs is running")

	go func() {
		if bootstrapNodes := options.BootstrapNodes; bootstrapNodes != nil && len(bootstrapNodes) != 0 {
			if err = i.ConnectToPeers(ctx, bootstrapNodes); err != nil {
				log.Errorf("Failed connect to peers: %s", err)
			}
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Infof("Shutting down IPFS ipfs")
				if err = i.node.PeerHost.Close(); err != nil {
					log.Errorf("Error: %s encountered shutting down IPFS", err)
				}
				os.Exit(0)
			}
		}
	}()

	if d, err := db.DB(i.IpfsOptions.DBPath, ctx); err != nil {
		return nil, err
	} else {
		i.FileApi = NewFileApi(i, d)
	}

	return i, nil
}

func (i *Ipfs) FilesApi() *FileApi {
	return i.FileApi
}

func (i *Ipfs) CoreApi() *icore.CoreAPI {
	return i.CoreAPI
}

// Spawns a node on the default repo location, if the repo exists
func (i *Ipfs) spawnDefault(ctx context.Context) (*core.IpfsNode, error) {
	if err := i.setupPlugins(""); err != nil {
		return nil, err
	} else if defaultPath, err := i.getIpfsRepo(); err != nil {
		// shouldn't be possible
		return nil, err
	} else {
		return i.createNode(ctx, defaultPath)
	}
}

// Spawns a node to be used just for this run (i.e. creates a tmp repo)
func (i *Ipfs) spawnEphemeral(ctx context.Context) (*core.IpfsNode, error) {
	if err := i.setupPlugins(""); err != nil {
		return nil, err
	}

	// Create a Temporary Repo
	if repoPath, err := i.getIpfsRepo(); err != nil {
		return nil, fmt.Errorf("failed to create temp repo: %s", err)
	} else {
		// Spawning an ephemeral IPFS ipfs
		return i.createNode(ctx, repoPath)
	}
}

/// ------ Setting up the IPFS Repo

func (i *Ipfs) setupPlugins(externalPluginsPath string) error {
	// Load any external plugins if available on externalPluginsPath
	plugins, err := loader.NewPluginLoader(filepath.Join(externalPluginsPath, "plugins"))
	if err != nil {
		return fmt.Errorf("error loading plugins: %s", err)
	}

	// Load preloaded and external plugins
	if err = plugins.Initialize(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	if err = plugins.Inject(); err != nil {
		return fmt.Errorf("error initializing plugins: %s", err)
	}

	return nil
}

func (i *Ipfs) getIpfsRepo() (string, error) {
	var (
		repoPath string
		err      error
		initRepo bool
		cfg      *config.Config
		//ipfscfg  *ipfsconfig.Config
	)

	if i.NodeType == ipfsconfig.Ephemeral {
		if repoPath, err = os.MkdirTemp("", "ipfs-shell"); err != nil {
			return "", fmt.Errorf("failed to get temp dir: %s", err)
		} else {
			initRepo = true
		}
	} else if i.NodeType == ipfsconfig.Default {
		if repoPath, err = config.PathRoot(); err != nil {
			// shouldn't be possible
			return "", err
		} else if _, err = os.Stat(repoPath); err != nil {
			if err = os.Mkdir(repoPath, 0700); err != nil {
				log.Error(err)
			} else {
				initRepo = true
			}
		}
	}

	// Create a config with default options and a 2048 bit key
	if initRepo {
		if cfg, err = config.Init(io.Discard, 2048); err != nil {
			return "", err
		}
	}

	// When creating the repository, you can define custom settings on the repository, such as enabling experimental
	// features (See experimental-features.md) or customizing the gateway endpoint.
	// To do such things, you should modify the variable `cfg`. For example:
	if i.EnableExperimental && initRepo {
		// https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#ipfs-filestore
		cfg.Experimental.FilestoreEnabled = true
		// https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#ipfs-urlstore
		cfg.Experimental.UrlstoreEnabled = true
		// https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#directory-sharding--hamt
		cfg.Experimental.ShardingEnabled = true
		// https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#ipfs-p2p
		cfg.Experimental.Libp2pStreamMounting = true
		// https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#p2p-http-proxy
		cfg.Experimental.P2pHttpProxy = true
		// https://github.com/ipfs/go-ipfs/blob/master/docs/experimental-features.md#strategic-providing
		cfg.Experimental.StrategicProviding = true
	}

	// Create the repo with the config
	if initRepo {
		if err = fsrepo.Init(repoPath, cfg); err != nil {
			return "", fmt.Errorf("failed to init ipfs: %s", err)
		}
	}

	return repoPath, nil
}

/// ------ Spawning the ipfs

// Creates an IPFS node and returns its coreAPI
func (i *Ipfs) createNode(ctx context.Context, repoPath string) (*core.IpfsNode, error) {
	// Open the repo
	repo, err := fsrepo.Open(repoPath)
	if err != nil {
		return nil, err
	}

	// Construct the ipfs

	nodeOptions := &core.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption, // This option sets the ipfs to be a full DHT ipfs (both fetching and storing DHT Records)
		// Routing: libp2p.DHTClientOption, // This option sets the ipfs to be a client DHT ipfs (only fetching records)
		Repo: repo,
	}

	node, err := core.NewNode(ctx, nodeOptions)
	if err != nil {
		return nil, err
	}
	node.IsDaemon = true

	return node, nil
}

func (i *Ipfs) ConnectToPeers(ctx context.Context, peers []string) error {
	log.Info("-- Going to connect to a few nodes in the Network as bootstrap peers --")
	var wg sync.WaitGroup
	peerInfos := make(map[peer.ID]*peer.AddrInfo, len(peers))
	for _, addrStr := range peers {
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		pii, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return err
		}
		pi, ok := peerInfos[pii.ID]
		if !ok {
			pi = &peer.AddrInfo{ID: pii.ID}
			peerInfos[pi.ID] = pi
		}
		pi.Addrs = append(pi.Addrs, pii.Addrs...)
	}

	log.Infof("Adding %d bootstrap peers", len(peerInfos))
	wg.Add(len(peerInfos))
	swarm := (*i.CoreAPI).Swarm()
	for _, peerInfo := range peerInfos {
		go func(peerInfo *peer.AddrInfo) {
			defer wg.Done()
			err := swarm.Connect(ctx, *peerInfo)
			if err != nil {
				log.Warnf("Failed to connect to %s: %s", peerInfo.ID.String(), err)
			}
		}(peerInfo)
	}
	wg.Wait()

	go func() {
		select {
		case <-time.After(10 * time.Second):
			if connections, err := swarm.Peers(ctx); err != nil {
				log.Error(err)
			} else {
				for _, connection := range connections {
					var latency time.Duration
					if latency, err = connection.Latency(); err != nil {
						log.Error(err)
					}
					log.Infof("Connected to %s (%s) Latency %d ms", connection.ID().String(), connection.Address().String(), latency.Milliseconds())
				}
			}

			if listenAddrs, err := swarm.LocalAddrs(ctx); err != nil {
				log.Error(err)
			} else {
				for _, addr := range listenAddrs {
					log.Infof("Swarm listening on %s", addr.String())
				}
			}
		}
	}()

	return nil
}
