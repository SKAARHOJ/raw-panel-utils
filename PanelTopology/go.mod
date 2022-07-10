module PanelTopology

go 1.18

require (
	github.com/SKAARHOJ/ibeam-lib-utils v1.0.0
	github.com/SKAARHOJ/rawpanel-lib v1.0.1
	github.com/SKAARHOJ/rawpanel-lib/topology v0.0.0-20220707120132-df79fc8fa986
	github.com/gorilla/websocket v1.5.0
	github.com/s00500/env_logger v0.1.24
	golang.org/x/exp v0.0.0-20220706164943-b4a6d9510983
	google.golang.org/protobuf v1.28.0
)

require (
	github.com/antchfx/xpath v0.0.0-20170515025933-1f3266e77307 // indirect
	github.com/disintegration/gift v1.2.1 // indirect
	github.com/fogleman/gg v1.3.0 // indirect
	github.com/golang/freetype v0.0.0-20170609003504-e2365dfdc4a0 // indirect
	github.com/mattn/go-colorable v0.1.9 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/subchen/go-xmldom v1.1.2 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	golang.org/x/image v0.0.0-20220617043117-41969df76e82 // indirect
	golang.org/x/sys v0.0.0-20220422013727-9388b58f7150 // indirect
)

replace github.com/SKAARHOJ/rawpanel-lib => ../../rawpanel-lib

replace github.com/SKAARHOJ/rawpanel-lib/topology => ../../rawpanel-lib/topology
