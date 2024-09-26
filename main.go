package main

import (
	"github.com/kairos-io/kairos/provider-kubeadm/pkg/provider"

	"github.com/kairos-io/kairos-sdk/clusterplugin"
	"github.com/sirupsen/logrus"
)

func main() {
	plugin := clusterplugin.ClusterPlugin{
		Provider: provider.ClusterProvider,
	}

	if err := plugin.Run(); err != nil {
		logrus.Fatal(err)
	}
}
