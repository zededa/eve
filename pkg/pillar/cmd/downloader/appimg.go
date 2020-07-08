// Copyright (c) 2019-2020 Zededa, Inc.
// SPDX-License-Identifier: Apache-2.0

package downloader

import (
	"github.com/lf-edge/eve/pkg/pillar/types"
)

// for function name consistency
func handleAppImgModify(ctxArg interface{}, key string,
	configArg interface{}) {

	dHandler.modify(ctxArg, types.AppImgObj, key, configArg)
}

func handleAppImgCreate(ctxArg interface{}, key string,
	configArg interface{}) {

	dHandler.create(ctxArg, types.AppImgObj, key, configArg)
}

func handleAppImgDelete(ctxArg interface{}, key string, configArg interface{}) {
	dHandler.delete(ctxArg, key, configArg)
}
