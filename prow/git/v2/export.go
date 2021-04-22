package git

import (
	"fmt"
	"os"
	"path"
)

func NewClientFactoryOnPV(opt ClientFactoryOpt) (ClientFactory, error) {
	option := &ClientFactoryOpts{}
	opt(option)

	cacheDir := path.Join(*option.CacheDirBase, option.Host, "gitcache")
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
			return nil, err
		}
	}

	f, err := NewClientFactory(opt)
	if err != nil {
		return nil, err
	}

	c, ok := f.(*clientFactory)
	if !ok {
		return nil, fmt.Errorf("Error to convert git client factory")
	}

	old := c.cacheDir
	c.cacheDir = cacheDir
	os.RemoveAll(old)

	return c, nil
}
