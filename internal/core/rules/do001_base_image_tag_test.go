package rules

import "testing"

func TestDO001(t *testing.T) {
	cases := map[string]int{
		"FROM alpine\n":                           1,
		"FROM alpine:latest\n":                    1,
		"FROM localhost:5000/app\n":               1,
		"FROM alpine:3.20\n":                      0,
		"FROM localhost:5000/app:1.2\n":           0,
		"FROM scratch\n":                          0,
		"ARG V=1\nFROM golang:${V}\n":             0,
		"FROM alpine@sha256:abcd\n":               0,
		"FROM golang:1.22 AS build\nFROM build\n": 0,
	}
	for src, n := range cases {
		t.Run(src, func(t *testing.T) {
			assertFindings(t, "DO001", ctxFor(t, src), n)
		})
	}
}
