package main

import "errors"

// Commands that are on their way.

func (a *app) audit(args []string) (int, error) { return 0, errors.New("audit is not implemented yet") }

func (a *app) bench(args []string) (int, error) { return 0, errors.New("bench is not implemented yet") }

func (a *app) ci(args []string) (int, error) { return 0, errors.New("ci is not implemented yet") }
