// Package model holds dockopt's own representation of a Dockerfile.
//
// The parser converts buildkit's node tree into these types instead of
// using buildkit's AST directly, because that AST is awkward to mutate.
// Rules read this model and fixers change it. The renderer then writes it
// back, keeping the original raw text of instructions nobody touched.
//
// Like everything under internal/core, this package must stay pure:
// no I/O, no docker, no filesystem access (it also has to build for wasm).
package model
