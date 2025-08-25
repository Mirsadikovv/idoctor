//go:build !dev
// +build !dev

package main

import (
	src "git.sriss.uz/mehnat/dmi_bot/app"
	"git.sriss.uz/shared/shared_service/sharedutil"
)

func fillEnv(env *src.Env) {
	sharedutil.Load(env)
}
