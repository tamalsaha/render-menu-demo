package main

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/discovery"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RenderMenu(kc client.Client, disco discovery.ServerResourcesInterface, obj runtime.Object, _ rest.ValidateObjectFunc, _ *metav1.CreateOptions) (runtime.Object, error) {
	in := obj.(*rsapi.RenderMenu)
	if in.Request == nil {
		return nil, apierrors.NewBadRequest("missing apirequest")
	}
	req := in.Request

	switch req.Mode {
	case rsapi.MenuAccordion:
		if menu, err := RenderAccordionMenu(kc, disco, req.Menu); err != nil {
			return nil, err
		} else {
			in.Response = menu
		}
	case rsapi.MenuGallery:
		if menu, err := RenderGalleryMenu(kc, disco, req.Menu); err != nil {
			return nil, err
		} else {
			in.Response = menu
		}
	case rsapi.MenuDropDown:
		if menu, err := RenderDropDownMenu(kc, disco, req); err != nil {
			return nil, err
		} else {
			in.Response = menu
		}
	default:
		return nil, apierrors.NewBadRequest(fmt.Sprintf("unknown menu mode %s", req.Mode))
	}
	return in, nil
}
