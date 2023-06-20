module github.com/tsuru/tsuru-client

go 1.12

require (
	github.com/Masterminds/semver/v3 v3.1.1
	github.com/ajg/form v0.0.0-20160822230020-523a5da1a92f
	github.com/andrestc/docker-machine-driver-cloudstack v0.9.2 // indirect
	github.com/antihax/optional v1.0.0
	github.com/cenkalti/backoff v0.0.0-20160904140958-8edc80b07f38 // indirect
	github.com/codegangsta/cli v1.19.1 // indirect
	github.com/digitalocean/godo v1.1.1 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/machine v0.16.1
	github.com/exoscale/egoscale v0.9.31 // indirect
	github.com/fsouza/go-dockerclient v1.7.4
	github.com/garyburd/redigo v1.6.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-openapi/validate v0.19.5 // indirect
	github.com/iancoleman/orderedmap v0.2.0
	github.com/intel-go/cpuid v0.0.0-20181003105527-1a4a6f06a1c6 // indirect
	github.com/jinzhu/copier v0.0.0-20180308034124-7e38e58719c3 // indirect
	github.com/mattn/go-shellwords v1.0.12
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/pkg/errors v0.9.1
	github.com/pmorie/go-open-service-broker-client v0.0.0-20180330214919-dca737037ce6
	github.com/rackspace/gophercloud v0.0.0-20160825135439-c90cb954266e // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20171017070213-362f9845770f
	github.com/samalba/dockerclient v0.0.0-20160531175551-a30362618471 // indirect
	github.com/tent/http-link-go v0.0.0-20130702225549-ac974c61c2f9 // indirect
	github.com/tsuru/docker-cluster v0.0.0-20190325123005-f372d8d4e354 // indirect
	github.com/tsuru/gnuflag v0.0.0-20151217162021-86b8c1b864aa
	github.com/tsuru/go-tsuruclient v0.0.0-20230612145111-83c76176241f
	github.com/tsuru/tablecli v0.0.0-20190131152944-7ded8a3383c6
	github.com/tsuru/tsuru v0.0.0-20230619203800-4d111c8b1584
	github.com/vmware/govcloudair v0.0.2 // indirect
	gopkg.in/amz.v3 v3.0.0-20161215130849-8c3190dff075 // indirect
	gopkg.in/bsm/ratelimit.v1 v1.0.0-20160220154919-db14e161995a // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c
	gopkg.in/redis.v3 v3.6.4 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/apimachinery v0.23.4
	launchpad.net/gocheck v0.0.0-20140225173054-000000000087 // indirect
)

replace (
	github.com/ajg/form => github.com/cezarsa/form v0.0.0-20210510165411-863b166467b9
	github.com/samalba/dockerclient => github.com/cezarsa/dockerclient v0.0.0-20190924055524-af5052a88081
)
