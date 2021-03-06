package main

import (
	"sort"
	"strings"

	"github.com/gobuffalo/flect"
	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	kmapi "kmodules.xyz/client-go/api/v1"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
	"kmodules.xyz/resource-metadata/hub/resourceeditors"
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
		ObjectMeta: mo.ObjectMeta,
		Spec: rsapi.MenuSpec{
			Mode: rsapi.MenuAccordion,
			Home: mo.Spec.Home.ToMenuSectionInfo(),
		},
	}
	menu.UID = types.UID(uuid.Must(uuid.NewUUID()).String()) // needed to save menu in configmap

	reg := hub.NewRegistryOfKnownResources()

	for _, so := range mo.Spec.Sections {
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
						if len(mi.Icons) == 0 {
							mi.Icons = generated.Icons
						}
					} else if gvr, ok := reg.FindGVR(item.Type, true); ok {
						rd, _ := reg.LoadByGVR(gvr)
						ed, _ := resourceeditors.LoadByGVR(kc, gvr)

						mi.Resource = &rd.Spec.Resource
						mi.Missing = true
						mi.Icons = rd.Spec.Icons
						mi.Installer = ed.Spec.Installer
						// mi.LayoutName = ""
					} else {
						mi.Resource = &kmapi.ResourceID{
							Group:   item.Type.Group,
							Version: "v1alpha1",                                       // fake default
							Name:    strings.ToLower(flect.Pluralize(item.Type.Kind)), // fake resource name
							Kind:    item.Type.Kind,
							Scope:   kmapi.NamespaceScoped, // fake default
						}
						mi.Icons = []rsapi.ImageSpec{
							{
								Source: hub.CRDIconSVG,
								Size:   "",
								Type:   "image/svg+xml",
							},
						}
						mi.Missing = true
					}
				}
				items = append(items, mi)
			}
			sec.Items = items
		}

		if len(sec.Items) > 0 {
			menu.Spec.Sections = append(menu.Spec.Sections, &sec)
		}
	}

	return &menu, nil
}
