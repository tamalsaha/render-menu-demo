package main

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
)

func CC(client discovery.ServerResourcesInterface) (*v1alpha1.Menu, error) {
	reg := hub.NewRegistryOfKnownResources()

	rsLists, err := client.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}

	mp := map[string]*v1alpha1.MenuSection{}
	for _, rsList := range rsLists {
		gv, err := schema.ParseGroupVersion(rsList.GroupVersion)
		if err != nil {
			return nil, err
		}

		sec := v1alpha1.MenuSection{
			Name: menuoutlines.MenuSectionName(gv.Group),
			ResourceClassInfo: v1alpha1.ResourceClassInfo{
				APIGroup:    gv.Group,
				Icons:       nil,
				Maintainers: nil,
				Links:       nil,
			},
			Items: nil,
		}

		for _, rs := range rsList.APIResources {
			gvr := gv.WithResource(rs.Name)
			rd, err := reg.LoadByGVR(gvr)
			if err != nil {
				if hub.IsUnregisteredErr(err) {

				} else {
					return nil, err
				}
			}
		}
	}

	// menuoutlines.MenuSectionName()

}
