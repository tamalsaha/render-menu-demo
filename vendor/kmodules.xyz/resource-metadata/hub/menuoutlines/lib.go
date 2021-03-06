/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package menuoutlines

import (
	"embed"
	iofs "io/fs"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"

	"github.com/gobuffalo/flect"
	"github.com/pkg/errors"
	"golang.org/x/net/publicsuffix"
	ioutilx "gomodules.xyz/x/ioutil"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/yaml"
)

var (
	//go:embed **/*.yaml trigger
	fs embed.FS

	m     sync.Mutex
	moMap map[string]*v1alpha1.MenuOutline

	loader = ioutilx.NewReloader(
		filepath.Join("/tmp", "hub", "menuoutlines"),
		fs,
		func(fsys iofs.FS) {
			moMap = map[string]*v1alpha1.MenuOutline{}

			if err := iofs.WalkDir(fsys, ".", func(path string, d iofs.DirEntry, err error) error {
				if d.IsDir() || err != nil {
					return errors.Wrap(err, path)
				}
				ext := filepath.Ext(d.Name())
				if ext != ".yaml" && ext != ".yml" && ext != ".json" {
					return nil
				}

				data, err := iofs.ReadFile(fsys, path)
				if err != nil {
					return errors.Wrap(err, path)
				}
				var obj v1alpha1.MenuOutline
				err = yaml.Unmarshal(data, &obj)
				if err != nil {
					return errors.Wrap(err, path)
				}
				moMap[obj.Name] = &obj
				return nil
			}); err != nil {
				panic(errors.Wrapf(err, "failed to load %s", reflect.TypeOf(v1alpha1.MenuOutline{})))
			}
		},
	)
)

func init() {
	loader.ReloadIfTriggered()
}

func EmbeddedFS() iofs.FS {
	return fs
}

func MenuSectionName(apiGroup string) string {
	name := apiGroup
	name = strings.TrimSuffix(name, ".k8s.io")
	name = strings.TrimSuffix(name, ".x-k8s.io")

	idx := strings.IndexRune(name, '.')
	if idx >= 0 {
		eTLD, icann := publicsuffix.PublicSuffix(name)
		if icann {
			name = strings.TrimSuffix(name, "."+eTLD)
		}
		parts := strings.Split(name, ".")
		for i := 0; i < len(parts)/2; i++ {
			j := len(parts) - i - 1
			parts[i], parts[j] = parts[j], parts[i]
		}
		name = strings.Join(parts, " ")
	}
	if name != "" {
		name = flect.Titleize(flect.Humanize(flect.Singularize(name)))
	} else {
		name = "Core"
	}
	return name
}

func LoadByName(name string) (*v1alpha1.MenuOutline, error) {
	m.Lock()
	defer m.Unlock()
	loader.ReloadIfTriggered()

	if obj, ok := moMap[name]; ok {
		return obj, nil
	}
	return nil, apierrors.NewNotFound(v1alpha1.Resource(v1alpha1.ResourceKindMenuOutline), name)
}

func List() []v1alpha1.MenuOutline {
	m.Lock()
	defer m.Unlock()
	loader.ReloadIfTriggered()

	out := make([]v1alpha1.MenuOutline, 0, len(moMap))
	for _, rl := range moMap {
		out = append(out, *rl)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func Names() []string {
	m.Lock()
	defer m.Unlock()
	loader.ReloadIfTriggered()

	out := make([]string, 0, len(moMap))
	for name := range moMap {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
