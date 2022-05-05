// Package config defines all the variables that users can set (through flags,
// environment vars, or the config file) to affect command behavior.
//
// The variables are grouped into a hierarchy of structs that matches the
// subcommand tree. Each struct defines destination fields and the mapping from
// command-specific flags into those fields.
//
// These structs are defined in a separate package from the actual behavior
// (in the 'internal/command' package) so that nested commands can refer to the
// "inherited" config from parent commands without creating an import loop.
package config
