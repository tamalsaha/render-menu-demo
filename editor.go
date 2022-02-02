package main

import (
	"context"
	"fmt"

	kmapi "kmodules.xyz/client-go/api/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog/v2"
	rsapi "kmodules.xyz/resource-metadata/apis/meta/v1alpha1"
	"kmodules.xyz/resource-metadata/hub/resourceeditors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func LoadResourceEditor(kc client.Client, gvr schema.GroupVersionResource) (*rsapi.ResourceEditor, bool) {
	var ed rsapi.ResourceEditor
	err := kc.Get(context.TODO(), client.ObjectKey{Name: resourceeditors.DefaultEditorName(gvr)}, &ed)
	if err == nil {
		return &ed, true
	} else if client.IgnoreNotFound(err) != nil {
		klog.V(8).InfoS(fmt.Sprintf("failed to load resource editor for %+v", gvr))
	}
	return resourceeditors.LoadForGVR(gvr)
}

func getEditor(rid *kmapi.ResourceID) (*rsapi.ResourceEditor, bool) {
	if rid == nil {
		return nil, false
	}

	gvr := rid.GroupVersionResource()
	ed, ok := resourceeditors.LoadForGVR(gvr)
	return ed, ok
}
