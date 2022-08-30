package main

import (
	"context"
	"log"
	"runtime"

	"github.com/getlantern/systray"
	"github.com/skratchdot/open-golang/open"
	"jonwillia.ms/biketray/geo"
)

func MenuItemLocation(ctx context.Context, geoMgr *geo.Manager) {

	mi := systray.AddMenuItem("🌎 Getting location", "")
	mi.Disable()
	mi.Hide()

	var openF func()
	switch runtime.GOOS {
	case "darwin":
		openF = openLocSettingsDarwin
	}
	if openF != nil {
		mi.Enable()
	}

	go func() {
		c := geoMgr.Subscribe(true)
		defer geoMgr.Unsubscribe(c)
		for {
			select {
			case <-ctx.Done():
				return
			case <-mi.ClickedCh:
				openF()
			case loc := <-c:
				if loc.Error != nil {
					mi.Show()
					mi.SetTitle("⚠️ Unable to get location, check settings")
					mi.SetTooltip(loc.Error.Error())
				} else {
					mi.Hide()
				}
			}
		}
	}()
}

func openLocSettingsDarwin() {
	err := open.Start("x-apple.systempreferences:com.apple.preference.security?Privacy_LocationServices")
	if err != nil {
		log.Println("problem opening location settings", err)
	}
}
