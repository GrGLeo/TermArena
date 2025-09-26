module github.com/GrGLeo/TermArena/test/e2e

go 1.23.2

require github.com/GrGLeo/TermArena/pkg/shared v0.0.0

replace github.com/GrGLeo/TermArena/client => ../../client

replace github.com/GrGLeo/TermArena/server => ../../server

replace github.com/GrGLeo/TermArena/pkg/shared => ../../pkg/shared
