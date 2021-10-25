package ipfs

import (
	"context"
	"fmt"
	"github.com/ipfs/go-cidutil"
	files "github.com/ipfs/go-ipfs-files"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/rish1988/ipfs-files/log"
	"github.com/syndtr/goleveldb/leveldb"
	"io"
	"os"
	"strings"
)

type FileApi struct {
	name *string
	ipfs *Ipfs
	db   *leveldb.DB
}

func NewFileApi(ipfs *Ipfs, db *leveldb.DB) *FileApi {
	return &FileApi{
		ipfs: ipfs,
		db:   db,
	}
}

// RootPath returns IPFS path from the file name
func (f *FileApi) RootPath(fileName string) path.Path {
	parts := strings.Split(fileName, "/")
	if p, err := f.db.Get([]byte(parts[len(parts) - 1]), nil); err != nil {
		log.Warnf("Path for file name %s not found", fileName)
		return nil
	} else {
		return path.New(string(p))
	}
}

func (f *FileApi) AddFile(context context.Context, sourcePath string) (*File, error) {
	return f.AddDir(context, sourcePath)
}

func (f *FileApi) AddDir(context context.Context, sourceDirPath string) (*File, error) {
	someFile, err := f.getUnixfsNode(sourceDirPath)
	if err != nil {
		log.Errorf("Could not get FileApi: %s", err)
		return nil, err
	}

	cidFile, err := (*f.ipfs.CoreAPI).Unixfs().Add(context, someFile, func(settings *options.UnixfsAddSettings) error {
		settings.Pin = true
		settings.Progress = true
		return nil
	})

	if err != nil {
		log.Errorf("Could not add File %s. Reason %s", sourceDirPath, err)
		return nil, err
	}

	var cidFilePath path.Path = cidFile

	file := &File{
		namespace:      cidFile.Namespace(),
		cid:            cidFile.Cid().String(),
		root:           cidFile.Root().String(),
		pathWithPrefix: cidFile.String(),
		path:           &cidFilePath,
	}

	parts := strings.Split(sourceDirPath, "/")
	if err = f.db.Put([]byte(parts[len(parts) - 1]), []byte(file.pathWithPrefix), nil); err != nil {
		log.Warnf("Failed to cache key for File %s with CID %s", sourceDirPath, file.cid)
	} else {
		log.Infof("Cached CID %s for File %s", file.cid, sourceDirPath)
	}

	return file, nil
}

func (f *FileApi) getUnixfsFile(path string) (files.File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if file != nil {
			if err = file.Close(); err != nil {
				log.Error(err)
			}
		}
	}()

	st, err := file.Stat()
	if err != nil {
		return nil, err
	}

	fil, err := files.NewReaderPathFile(path, file, st)
	if err != nil {
		return nil, err
	}

	return fil, nil
}

func (f *FileApi) getUnixfsNode(path string) (files.Node, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	fil, err := files.NewSerialFile(path, false, st)
	if err != nil {
		return nil, err
	}

	return fil, nil
}

func (f *FileApi) GetFile(context context.Context, file path.Path) (*File, error) {
	if node, err := (*f.ipfs.CoreAPI).Unixfs().Get(context, file); err != nil {
		log.Errorf("Could not get file with CID: %s", err)
		return nil, err
	} else {
		_, _, _, cidString := cidutil.ScanForCid([]byte(file.String()))
		return &File{
			path: &file,
			node: &node,
			cid: cidString,
			namespace: file.Namespace(),
			pathWithPrefix: file.String(),
		}, nil
	}
}

func (f *FileApi) WalkDirTree(context context.Context, parent *File, walker DirWalker) error {
	nd, err := (*f.ipfs.CoreAPI).Unixfs().Get(context, *parent.path)
	if err != nil {
		log.Errorf("Could not get file with CID: %s", err)
		return err
	}

	child := &File{
		cid:            parent.cid,
		name:           parent.name,
		file:           parent.file,
		path:           parent.path,
		node:           parent.node,
		root:           parent.root,
		namespace:      parent.namespace,
		pathWithPrefix: parent.pathWithPrefix,
	}

	switch nd := nd.(type) {
	case *files.Symlink:
		child.Type = SymbolicLinkType
		child.target = nd.Target
		err = walker(child)
	case files.File:
		child.Type = FileType
		child.file = &nd
		err = walker(child)
	case files.Directory:
		child.Type = DirType
		if err = walker(child); err != nil {
			return err
		}

		entries := nd.Entries()
		for entries.Next() {
			node := entries.Node()
			name := entries.Name()

			child.node = &node

			dirPath := path.Join(*parent.path, name)

			child.name = name
			child.path = &dirPath
			child.pathWithPrefix = (*child.path).String()

			if err = f.WalkDirTree(context, child, walker); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("file type %T is not supported", nd)
	}
	return err
}

type File struct {
	Type

	path           *path.Path
	file           *files.File
	node           *files.Node
	name           string
	target         string
	namespace      string
	root           string
	cid            string
	pathWithPrefix string
}

type Type int

const (
	SymbolicLinkType Type = iota
	FileType
	DirType
)

type DirWalker func(file *File) error

func (f *File) Path() string {
	return strings.ReplaceAll(strings.ReplaceAll((*f.path).String(), f.Prefix(), "/"), "//", "/")
}

func (f *File) Name() string {
	if len(f.name) == 0 {
		return "/"
	}
	return f.name
}

func (f *File) Prefix() string {
	if len(f.root) != 0 {
		return fmt.Sprintf("/%s/%s", f.namespace, f.root)
	} else {
		return fmt.Sprintf("/%s/%s", f.namespace, f.cid)
	}
}

func (f *File) Read() ([]byte, error) {
	var (
		data     []byte
		err      error
		realLink string
	)

	switch f.Type {
	case SymbolicLinkType:
		if data, err = os.ReadFile(f.target); err != nil {
			log.Errorf("Error reading data from original symlinked file %s", realLink)
		}
	case FileType:
		if data, err = io.ReadAll(*f.file); err != nil {
			log.Errorf("Error reading symlinked file %s", f.name)
		}
	case DirType:
		err = fmt.Errorf("%s is a directory", (*f.path).String())
	default:
		err = fmt.Errorf("file type %s is not supported", (*f.path).String())
	}

	return data, err
}
