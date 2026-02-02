package memsize

// Sizes is a placeholder type to satisfy transitive dependencies.
//
// This repository only needs the memsize API surface for go-ethereum's optional
// debug endpoints. The original upstream implementation relies on runtime
// linkname hooks that are not compatible with newer Go toolchains.
type Sizes struct{}

// Scan is a no-op implementation that returns empty sizes.
func Scan(_ interface{}) Sizes { return Sizes{} }
