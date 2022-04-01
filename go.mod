module github.com/ffenix113/teleporter

go 1.18

require (
	github.com/Arman92/go-tdlib/v2 v2.0.0
	github.com/fsnotify/fsnotify v1.5.1
	github.com/go-chi/chi/v5 v5.0.7
	golang.org/x/exp v0.0.0-20220328175248-053ad81199eb
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b
)

require golang.org/x/sys v0.0.0-20211019181941-9d821ace8654 // indirect

replace github.com/Arman92/go-tdlib/v2 v2.0.0 => github.com/ffenix113/go-tdlib/v2 v2.0.0-20211204191913-dbb38e1deb80
