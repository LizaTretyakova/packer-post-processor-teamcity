package main

import (
	"fmt"

	"github.com/hashicorp/packer/common"
	"github.com/hashicorp/packer/helper/config"
	"github.com/hashicorp/packer/packer"
	"github.com/hashicorp/packer/packer/plugin"
	"github.com/hashicorp/packer/template/interpolate"
	"os"
	"strings"
)

const TeamcityVersionEnvVar = "TEAMCITY_VERSION"
var AmazonBuilderIds = []string {
	"mitchellh.amazonebs",
	"mitchellh.amazon.ebssurrogate",
	"mitchellh.amazon.instance",
	"mitchellh.amazon.chroot",
}

type Config struct {
	common.PackerConfig `mapstructure:",squash"`

	OutputPath string `mapstructure:"output"`
	StripPath  bool   `mapstructure:"strip_path"`
	ctx        interpolate.Context
}

type PostProcessor struct {
	config Config
}

type ManifestFile struct {
	Builds      []Artifact `json:"builds"`
	LastRunUUID string     `json:"last_run_uuid"`
}

func Contains(vs []string, t string) bool {
	for _, v := range vs {
		if v == t {
			return true
		}
	}
	return false
}

func (p *PostProcessor) Configure(raws ...interface{}) error {
	err := config.Decode(&p.config, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &p.config.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{},
		},
	}, raws...)
	if err != nil {
		return err
	}

	if p.config.OutputPath == "" {
		p.config.OutputPath = "packer-manifest.json"
	}

	if err = interpolate.Validate(p.config.OutputPath, &p.config.ctx); err != nil {
		return fmt.Errorf("Error parsing target template: %s", err)
	}

	return nil
}

func (p *PostProcessor) PostProcess(ui packer.Ui, source packer.Artifact) (packer.Artifact, bool, error) {
	if os.Getenv(TeamcityVersionEnvVar) != "" {
		if Contains(AmazonBuilderIds, source.BuilderId())  {
			s := strings.Split(source.Id(), ":")
			region, ami := s[0], s[1] // TODO: several AMIs
			ui.Message(fmt.Sprintf("##teamcity[setParameter name='packer.artifact.aws.region' value='%v']", region))
			ui.Message(fmt.Sprintf("##teamcity[setParameter name='packer.artifact.aws.ami' value='%v']", ami))
		} else {
			ui.Message(fmt.Sprintf("##teamcity[setParameter name='packer.artifact.id' value='%v']", source.Id()))
		}
	}
	return source, true, nil
}

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterPostProcessor(new(PostProcessor))
	server.Serve()
}
