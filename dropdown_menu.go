package main

import (
	"context"
	"fmt"
	gourl "net/url"
	"path"
	"sort"

	"github.com/pkg/errors"
	"gomodules.xyz/pointer"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/klog/v2"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub/menuoutlines"
	"kubepack.dev/kubepack/pkg/lib"
	chartsapi "kubepack.dev/preset/apis/charts/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RenderDropDownMenu(kc client.Client, disco discovery.ServerResourcesInterface, opts *rsapi.RenderMenuRequest) (*rsapi.Menu, error) {
	mo, err := menuoutlines.LoadByName(opts.Menu)
	if err != nil {
		return nil, err
	}

	menuPerGK, err := GenerateMenuItems(kc, disco)
	if err != nil {
		return nil, err
	}

	out := rsapi.Menu{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rsapi.SchemeGroupVersion.String(),
			Kind:       rsapi.ResourceKindMenu,
		},
		Home:     mo.Home,
		Sections: nil,
	}

	for _, so := range mo.Sections {
		if opts.Section != nil && so.Name != *opts.Section {
			continue
		}

		sec := rsapi.MenuSection{
			MenuSectionInfo: so.MenuSectionInfo,
		}
		if sec.AutoDiscoverAPIGroup != "" {
			kinds := menuPerGK[sec.AutoDiscoverAPIGroup]
			for kind, item := range kinds {
				if opts.Type != nil &&
					(opts.Type.Group != sec.AutoDiscoverAPIGroup || opts.Type.Kind != kind) {
					continue
				}
				sec.Items = append(sec.Items, *item) // variants
			}
		} else {
			items := make([]rsapi.MenuItem, 0)
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
					if generated, ok := getMenuItem(menuPerGK, *item.Type); ok {
						mi.Resource = generated.Resource
						mi.Missing = false
						mi.Installer = generated.Installer
						if mi.LayoutName == "" {
							mi.LayoutName = generated.LayoutName
						}
					}
				}

				if mi.Resource != nil &&
					opts.Type != nil &&
					(opts.Type.Group != mi.Resource.Group || opts.Type.Kind != mi.Resource.Kind) {
					continue
				}

				ed, ok := getEditor(mi.Resource)
				if !ok || ed.Spec.UI == nil || ed.Spec.UI.Options == nil || len(ed.Spec.Variants) == 0 {
					items = append(items, mi)
				} else if mi.Resource != nil {
					gvr := mi.Resource.GroupVersionResource()
					ed, ok := LoadResourceEditor(kc, gvr)
					if !ok {
						return nil, fmt.Errorf("ResourceEditor not defined for %+v", gvr)
					}

					chartRef := ed.Spec.UI.Options
					chrt, err := lib.DefaultRegistry.GetChart(chartRef.URL, chartRef.Name, chartRef.Version)
					if err != nil {
						klog.Fatal(err)
					}

					vpsMap, err := LoadVendorPresets(chrt)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to load vendor presets for chart %+v", chartRef)
					}

					for _, ref := range ed.Spec.Variants {
						if ref.APIGroup == nil {
							ref.APIGroup = pointer.StringP(chartsapi.GroupVersion.Group)
						}
						if ref.Kind != chartsapi.ResourceKindVendorChartPreset && ref.Kind != chartsapi.ResourceKindClusterChartPreset {
							return nil, fmt.Errorf("unknown preset kind %q used in menu item %s", ref.Kind, mi.Name)
						}

						qs := gourl.Values{}
						qs.Set("preset-group", *ref.APIGroup)
						qs.Set("preset-kind", ref.Kind)
						qs.Set("preset-name", ref.Name)
						u := gourl.URL{
							Path:     path.Join(mi.Resource.Group, mi.Resource.Version, mi.Resource.Name),
							RawQuery: qs.Encode(),
						}

						name, err := GetPresetName(kc, chartRef, vpsMap, ref)
						if err != nil {
							return nil, err
						}

						if len(ed.Spec.Variants) == 1 {
							// cp := mi
							mi.Name = name
							mi.Path = u.String()
							mi.Preset = &ref
							// items = append(items, mi)
						} else {
							cp := mi
							cp.Name = name
							cp.Path = u.String()
							cp.Preset = &ref
							mi.Items = append(mi.Items, cp)
						}
					}
					items = append(items, mi)
				}
			}
			sec.Items = items
		}
		sort.Slice(sec.Items, func(i, j int) bool {
			return sec.Items[i].Name < sec.Items[j].Name
		})

		if len(sec.Items) > 0 {
			out.Sections = append(out.Sections, &sec)
		}
	}

	return &out, nil
}

func GetPresetName(
	kc client.Client,
	chartRef *rsapi.ChartRepoRef,
	vpsMap map[string]*chartsapi.VendorChartPreset,
	ref core.TypedLocalObjectReference) (string, error) {
	if ref.Kind == chartsapi.ResourceKindVendorChartPreset {
		ps, ok := vpsMap[ref.Name]
		if !ok {
			return "", fmt.Errorf("%s %s not found in chart %+v", chartsapi.ResourceKindVendorChartPreset, ref.Name, chartRef)
		}
		return ps.Name, nil
	}

	var ps chartsapi.ClusterChartPreset
	err := kc.Get(context.TODO(), client.ObjectKey{Name: ref.Name}, &ps)
	if err != nil {
		return "", errors.Wrapf(err, "%s %s not found", chartsapi.ResourceKindClusterChartPreset, ref.Name)
	}
	return ps.Name, nil
}
