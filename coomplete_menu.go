package main

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
	"kmodules.xyz/resource-metadata/hub/resourceeditors"
	"kmodules.xyz/resource-metadata/hub/resourceoutlines"
	"sort"
	"strings"
)

var defaultIcons = []v1alpha1.ImageSpec{
	{
		Source: crdIconSVG,
		Type:   "image/svg+xml",
	},
}

func GenerateCompleteMenu(client discovery.ServerResourcesInterface) (*v1alpha1.Menu, error) {
	sectionIcons := map[string][]v1alpha1.ImageSpec{}
	for _, m := range menuoutlines.List() {
		for _, sec := range m.Spec.Sections {
			if sec.AutoDiscoverAPIGroup != "" {
				sectionIcons[sec.AutoDiscoverAPIGroup] = sec.Icons
			}
		}
	}

	reg := hub.NewRegistryOfKnownResources()

	rsLists, err := client.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}

	sections := make([]*v1alpha1.MenuSection, 0, len(rsLists))
	for _, rsList := range rsLists {
		gv, err := schema.ParseGroupVersion(rsList.GroupVersion)
		if err != nil {
			return nil, err
		}

		sec := v1alpha1.MenuSection{
			Name: menuoutlines.MenuSectionName(gv.Group),
		}
		if icons, ok := sectionIcons[gv.Group]; ok {
			sec.Icons = icons
		} else {
			sec.Icons = defaultIcons
		}

		for _, rs := range rsList.APIResources {
			// skip sub resource
			if strings.ContainsRune(rs.Name, '/') {
				continue
			}

			// if resource can't be listed or read (get) or only view type skip it
			verbs := sets.NewString(rs.Verbs...)
			if !verbs.HasAll("list", "get", "watch", "create") {
				continue
			}

			scope := kmapi.ClusterScoped
			if rs.Namespaced {
				scope = kmapi.NamespaceScoped
			}
			rid := kmapi.ResourceID{
				Group:   gv.Group,
				Version: gv.Version,
				Name:    rs.Name,
				Kind:    rs.Kind,
				Scope:   scope,
			}
			gvr := rid.GroupVersionResource()

			me := v1alpha1.MenuItem{
				Name:       rid.Kind,
				Path:       "",
				Resource:   &rid,
				Missing:    false,
				Required:   false,
				LayoutName: resourceoutlines.DefaultLayoutName(gvr),
				// Icons:    rd.Spec.Icons,
				// Installer:  rd.Spec.Installer,
			}
			if rd, err := reg.LoadByGVR(gvr); err == nil {
				me.Icons = rd.Spec.Icons
			}
			if rd, ok := resourceeditors.LoadForGVR(gvr); ok {
				me.Installer = rd.Spec.Installer
			}

			sec.Items = append(sec.Items, me) // variants
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

	return &v1alpha1.Menu{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.ResourceKindMenuOutline,
		},
		Sections: sections,
	}, nil
}
