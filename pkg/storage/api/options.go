package api

// Options client options
type Options struct {
	Mimetype string
}

// Option applier
type Option interface {
	Apply(o *Options)
}

// OptionFunc functional option implements
type OptionFunc func(o *Options)

// Apply apply option
func (of OptionFunc) Apply(o *Options) {
	of(o)
}

// NewOptions create options
func NewOptions() *Options {
	return &Options{}
}

// Apply apply options
func (o *Options) Apply(option ...Option) {
	for _, opt := range option {
		opt.Apply(o)
	}
}

// WithMimetype settings mimetype
func WithMimetype(mimetype string) Option {
	return OptionFunc(
		func(o *Options) {
			o.Mimetype = mimetype
		},
	)
}

