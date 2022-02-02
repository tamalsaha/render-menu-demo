package main

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RenderAccordionMenu(kc client.Client, disco discovery.ServerResourcesInterface, menuName string) (*rsapi.Menu, error) {
	mo, err := menuoutlines.LoadByName(menuName)
	if err != nil {
		return nil, err
	}

	out, err := GenerateMenuItems(kc, disco)
	if err != nil {
		return nil, err
	}

	menu := rsapi.Menu{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rsapi.SchemeGroupVersion.String(),
			Kind:       rsapi.ResourceKindMenu,
		},
		Home:     mo.Home.ToMenuSectionInfo(),
		Sections: nil,
	}

	for _, so := range mo.Sections {
		sec := rsapi.MenuSection{
			MenuSectionInfo: *so.MenuSectionOutlineInfo.ToMenuSectionInfo(),
		}
		if so.AutoDiscoverAPIGroup != "" {
			kinds := out[so.AutoDiscoverAPIGroup]
			for _, item := range kinds {
				sec.Items = append(sec.Items, *item) // variants
			}
			sort.Slice(sec.Items, func(i, j int) bool {
				return sec.Items[i].Name < sec.Items[j].Name
			})
		} else {
			items := make([]rsapi.MenuItem, 0, len(so.Items))
			for _, item := range so.Items {
				mi := rsapi.MenuItem{
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
				items = append(items, mi)
			}
			sec.Items = items
		}

		if len(sec.Items) > 0 {
			menu.Sections = append(menu.Sections, &sec)
		}
	}

	return &menu, nil
}
