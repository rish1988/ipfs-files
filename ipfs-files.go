package ipfs_files

import (
	"context"
	"github.com/rish1988/ipfs-files/config"
	"github.com/rish1988/ipfs-files/ipfs"
)

func NewIpfsNode(context context.Context, options config.IpfsOptions) (ipfs.IpfsApi, error) {
	return ipfs.StartIpfsNode(context, options)
}
