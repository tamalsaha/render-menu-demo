package main

import (
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
	kmapi "kmodules.xyz/client-go/api/v1"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub"
	"kmodules.xyz/resource-metadata/hub/resourceeditors"
	"kmodules.xyz/resource-metadata/hub/resourceoutlines"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RenderMenu(driver *UserMenuDriver, req *rsapi.RenderMenuRequest) (*rsapi.Menu, error) {
	switch req.Mode {
	case rsapi.MenuAccordion:
		return driver.Get(req.Menu)
	case rsapi.MenuGallery:
		return GetGalleryMenu(driver, req)
	case rsapi.MenuDropDown:
		return GetDropDownMenu(driver, req)
	default:
		return nil, apierrors.NewBadRequest(fmt.Sprintf("unknown menu mode %s", req.Mode))
	}
}

func GenerateMenuItems(kc client.Client, disco discovery.ServerResourcesInterface) (map[string]map[string]*rsapi.MenuItem, error) {
	reg := hub.NewRegistryOfKnownResources()

	rsLists, err := disco.ServerPreferredResources()
	if err != nil && !discovery.IsGroupDiscoveryFailedError(err) {
		return nil, err
	}

	// [group][Kind] => MenuItem
	out := map[string]map[string]*rsapi.MenuItem{}
	for _, rsList := range rsLists {
		gv, err := schema.ParseGroupVersion(rsList.GroupVersion)
		if err != nil {
			return nil, err
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

			me := rsapi.MenuItem{
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
			if rd, ok := resourceeditors.LoadByGVR(kc, gvr); ok {
				me.Installer = rd.Spec.Installer
			}

			if _, ok := out[gv.Group]; !ok {
				out[gv.Group] = map[string]*rsapi.MenuItem{}
			}
			out[gv.Group][rs.Kind] = &me // variants
		}
	}

	return out, nil
}

func getMenuItem(out map[string]map[string]*rsapi.MenuItem, gk metav1.GroupKind) (*rsapi.MenuItem, bool) {
	m, ok := out[gk.Group]
	if !ok {
		return nil, false
	}
	item, ok := m[gk.Kind]
	return item, ok
}
