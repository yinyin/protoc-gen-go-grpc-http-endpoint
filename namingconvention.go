package protocgenghe

import (
	nameconv "github.com/yinyin/go-convert-naming-convention"
)

type NamingConventionConverter interface {
	ConvertConvention(name string) string
}

type NoopNamingConventionConverter struct{}

func (ncc *NoopNamingConventionConverter) ConvertConvention(name string) string {
	return name
}

type ToKebabCase struct {
	Opts *nameconv.Options
}

func (ncc *ToKebabCase) ConvertConvention(name string) string {
	return nameconv.ToKebebCase(name, ncc.Opts)
}

type ToSnakeCase struct {
	Opts *nameconv.Options
}

func (ncc *ToSnakeCase) ConvertConvention(name string) string {
	return nameconv.ToSnakeCase(name, ncc.Opts)
}

type ToLowerCamelCase struct {
	Opts *nameconv.Options
}

func (ncc *ToLowerCamelCase) ConvertConvention(name string) string {
	return nameconv.ToLowerCamelCase(name, ncc.Opts)
}

type ToUpperCamelCase struct {
	Opts *nameconv.Options
}

func (ncc *ToUpperCamelCase) ConvertConvention(name string) string {
	return nameconv.ToUpperCamelCase(name, ncc.Opts)
}

func NewNamingConventionConverter(convention string, opts *nameconv.Options) NamingConventionConverter {
	switch convention {
	case "kebab-case":
		return &ToKebabCase{
			Opts: opts,
		}
	case "snake_case":
		return &ToSnakeCase{
			Opts: opts,
		}
	case "lowerCamelCase":
		return &ToLowerCamelCase{
			Opts: opts,
		}
	case "UpperCamelCase":
		return &ToUpperCamelCase{
			Opts: opts,
		}
	default:
		return &NoopNamingConventionConverter{}
	}
}
