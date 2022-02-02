package main

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
)

func RenderMenu(driver *UserMenuDriver, req *rsapi.RenderMenuRequest) (*rsapi.Menu, error) {
	switch req.Mode {
	case rsapi.MenuAccordion:
		return driver.Get(req.Menu)
	case rsapi.MenuGallery:
		return GetGalleryMenu(driver, req.Menu)
	case rsapi.MenuDropDown:
		return GetDropDownMenu(driver, req)
	default:
		return nil, apierrors.NewBadRequest(fmt.Sprintf("unknown menu mode %s", req.Mode))
	}
}
