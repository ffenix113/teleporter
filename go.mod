module github.com/ffenix113/teleporter

go 1.18

require (
	github.com/Arman92/go-tdlib/v2 v2.0.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/go-chi/chi/v5 v5.0.7
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c // indirect

replace github.com/Arman92/go-tdlib/v2 v2.0.0 => github.com/ffenix113/go-tdlib/v2 v2.0.0-20211204191913-dbb38e1deb80
