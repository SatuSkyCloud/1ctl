package version

// Environment is the target backend environment this binary was built for.
// Overridden at build time via -ldflags for the 1ctl-dev variant.
var Environment = "production"
