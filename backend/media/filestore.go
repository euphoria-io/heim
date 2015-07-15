package media

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"euphoria.io/heim/proto"
)

// type MediaStore interface {
// 	Get(mediaID string) (RawImage, error)

// 	Store(img RawImage) error
// }

func verifyPath(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(path, 0700); err != nil {
				return fmt.Errorf("mkdir %s: %s", path, err)
			}
			return nil
		}
		return fmt.Errorf("stat %s: %s", path, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

type LocalStore struct {
	root string
}

func NewLocalStore(root string) (*LocalStore, error) {
	if err := verifyPath(root); err != nil {
		return nil, err
	}
	return &LocalStore{root: root}, nil
}

func (ls *LocalStore) path(id string) string { return filepath.Join(ls.root, id[:2], id[2:4], id) }

func (ls *LocalStore) Store(id string, img *proto.RawImage) error {
	fpath := ls.path(id)
	if err := verifyPath(fpath); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(fpath, "img"), *img, 0600); err != nil {
		return err
	}
	return nil
}

func (ls *LocalStore) Get(mediaID string) (proto.RawImage, error) {
	_, err := os.Stat(filepath.Join(ls.path(mediaID), "img"))
	if err != nil {
		return nil, err
	}
	bytes, err := ioutil.ReadFile(filepath.Join(ls.path(mediaID), "img"))
	if err != nil {
		return nil, err
	}
	return proto.RawImage(bytes), nil
}
