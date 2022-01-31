package main

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
)

func RenderDropDownMenu(client discovery.ServerResourcesInterface, menuName string) (*v1alpha1.Menu, error) {
	mo, err := menuoutlines.LoadByName(menuName)
	if err != nil {
		return nil, err
	}

	out, err := GenerateMenuItems(client)
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
