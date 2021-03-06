package main

import (
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var defaultIcons = []rsapi.ImageSpec{
	{
		Source: hub.CRDIconSVG,
		Type:   "image/svg+xml",
	},
}

func GenerateCompleteMenu(kc client.Client, disco discovery.ServerResourcesInterface) (*rsapi.Menu, error) {
	sectionIcons := map[string][]rsapi.ImageSpec{}
	for _, m := range menuoutlines.List() {
		for _, sec := range m.Spec.Sections {
			if sec.AutoDiscoverAPIGroup != "" {
				sectionIcons[sec.AutoDiscoverAPIGroup] = sec.Icons
			}
		}
	}

	menuPerGK, err := GenerateMenuItems(kc, disco)
	if err != nil {
		return nil, err
	}

	sections := make([]*rsapi.MenuSection, 0, len(menuPerGK))
	for group, kinds := range menuPerGK {
		sec := rsapi.MenuSection{
			MenuSectionInfo: rsapi.MenuSectionInfo{
				Name: menuoutlines.MenuSectionName(group),
			},
		}
		if icons, ok := sectionIcons[group]; ok {
			sec.Icons = icons
		} else {
			sec.Icons = defaultIcons
		}

		for _, item := range kinds {
			sec.Items = append(sec.Items, *item) // variants
		}
		sort.Slice(sec.Items, func(i, j int) bool {
			return sec.Items[i].Name < sec.Items[j].Name
		})

		if len(sec.Items) > 0 {
			sections = append(sections, &sec)
		}
	}

	sort.Slice(sections, func(i, j int) bool {
		return sections[i].Name < sections[j].Name
	})

	return &rsapi.Menu{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rsapi.SchemeGroupVersion.String(),
			Kind:       rsapi.ResourceKindMenuOutline,
		},
		Spec: rsapi.MenuSpec{
			Sections: sections,
		},
	}, nil
}
