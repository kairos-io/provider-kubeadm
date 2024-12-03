package utils

import (
	yip "github.com/mudler/yip/pkg/schema"
)

func GetFileStage(stageName, path, content string) yip.Stage {
	return yip.Stage{
		Name: stageName,
		Files: []yip.File{
			{
				Path:        path,
				Permissions: 0640,
				Content:     content,
			},
		},
	}
}
