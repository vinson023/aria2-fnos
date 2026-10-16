//go:build !unix

package main

import "os"

func directoryOwner(_ os.FileInfo) (int, int, bool) { return -1, -1, false }

func chownDirectory(_ string, _, _ int) error { return nil }
