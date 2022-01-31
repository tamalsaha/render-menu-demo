package main

import (
	"fmt"
	"gomodules.xyz/pointer"
	"kmodules.xyz/resource-metadata/hub/resourceeditors"
	gourl "net/url"
	"path"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
	chartsapi "kubepack.dev/preset/apis/charts/v1alpha1"
)

func RenderGalleryMenu(client client.Client, disco discovery.ServerResourcesInterface, menuName string) (*v1alpha1.Menu, error) {
	mo, err := menuoutlines.LoadByName(menuName)
	if err != nil {
		return nil, err
	}

	out, err := GenerateMenuItems(disco)
	if err != nil {
		return nil, err
	}

	menu := v1alpha1.Menu{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.ResourceKindMenu,
		},
		Home:     mo.Home,
		Sections: nil,
	}

	for _, so := range mo.Sections {
		sec := v1alpha1.MenuSection{
			MenuSectionInfo: so.MenuSectionInfo,
		}
		if sec.AutoDiscoverAPIGroup != "" {
			kinds := out[sec.AutoDiscoverAPIGroup]
			for _, item := range kinds {
				sec.Items = append(sec.Items, *item) // variants
			}
			sort.Slice(sec.Items, func(i, j int) bool {
				return sec.Items[i].Name < sec.Items[j].Name
			})
		} else {
			items := make([]v1alpha1.MenuItem, 0, len(so.Items))
			for _, item := range so.Items {
				mi := v1alpha1.MenuItem{
					Name:       item.Name,
					Path:       item.Path,
					Resource:   nil,
					Missing:    true,
					Required:   item.Required,
					LayoutName: item.LayoutName,
					Icons:      item.Icons,
					Installer:  nil,
				}

				if item.Type != nil {
					if generated, ok := getMenuItem(out, *item.Type); ok {
						mi.Resource = generated.Resource
						mi.Missing = false
						mi.Installer = generated.Installer
						if mi.LayoutName == "" {
							mi.LayoutName = generated.LayoutName
						}
					}
				}

				if len(item.Variants) == 0 {
					items = append(items, mi)
				} else if mi.Resource != nil {
					gvr := mi.Resource.GroupVersionResource()
					ed, ok := resourceeditors.LoadForGVR(gvr)
					if !ok {
						return nil, fmt.Errorf("ResourceEditor not defined for %+v", gvr)
					}
					ed.Spec.UI

					for _, ref := range item.Variants {
						if ref.APIGroup == nil {
							ref.APIGroup = pointer.StringP(chartsapi.GroupVersion.Group)
						}
						if ref.Kind != chartsapi.ResourceKindVendorChartPreset || ref.Kind != chartsapi.ResourceKindClusterChartPreset {
							return nil, fmt.Errorf("unknown preset kind %q used in menu item %s", ref.Kind, mi.Name)
						}

						qs := gourl.Values{}
						qs.Set("preset-group", *ref.APIGroup)
						qs.Set("preset-kind", ref.Kind)
						qs.Set("preset-name", ref.Name)
						u2 := gourl.URL{
							Path:     path.Join(mi.Resource.Group, mi.Resource.Version, mi.Resource.Name),
							RawQuery: qs.Encode(),
						}

						if len(item.Variants) == 1 {
							// cp := mi
							mi.Name = ""
							mi.Path = u2.String()
							mi.Preset = &ref
							items = append(items, mi)
						} else {
							cp := mi
							cp.Name = ""
							cp.Path = u2.String()
							cp.Preset = &ref
							items = append(items, cp)
						}
					}
				}
			}
			sec.Items = items
		}

		if len(sec.Items) > 0 {
			menu.Sections = append(menu.Sections, &sec)
		}
	}

	return &menu, nil
}
