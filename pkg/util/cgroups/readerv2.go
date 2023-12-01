// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

//go:build linux

package cgroups

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/karrick/godirwalk"
)

const (
	controllersFile = "cgroup.controllers"
)

type readerV2 struct {
	cgroupRoot        string
	cgroupControllers map[string]struct{}
	filter            ReaderFilter
	pidMapper         pidMapper
	inodeMapper       map[uint64]Cgroup
}

func newReaderV2(procPath, cgroupRoot string, filter ReaderFilter) (*readerV2, error) {
	controllers, err := readCgroupControllers(cgroupRoot)
	if err != nil {
		return nil, err
	}

	return &readerV2{
		cgroupRoot:        cgroupRoot,
		cgroupControllers: controllers,
		filter:            filter,
		pidMapper:         getPidMapper(procPath, cgroupRoot, "", filter),
		inodeMapper:       make(map[uint64]Cgroup),
	}, nil
}

// cgroupByInode returns the cgroup for the given inode.
func (r *readerV2) cgroupByInode(inode uint64) (Cgroup, error) {
	cgroup, ok := r.inodeMapper[inode]
	if !ok {
		return nil, fmt.Errorf("no cgroup found for inode %d", inode)
	}
	return cgroup, nil
}

// parseCgroups parses the cgroups from the cgroupRoot and returns a map of cgroup id to cgroup.
// It also populates the inodeMapper.
func (r *readerV2) parseCgroups() (map[string]Cgroup, error) {
	res := make(map[string]Cgroup)

	err := godirwalk.Walk(r.cgroupRoot, &godirwalk.Options{
		AllowNonDirectory: true,
		Unsorted:          true,
		Callback: func(fullPath string, de *godirwalk.Dirent) error {
			if de.IsDir() {
				id, err := r.filter(fullPath, de.Name())
				if id != "" {
					relPath, err := filepath.Rel(r.cgroupRoot, fullPath)
					if err != nil {
						return err
					}
					inode, err := inodeForFile(fullPath)
					if err != nil {
						log.Debugf("failed to retrieve cgroup id for %s: %s", fullPath, err)
						return err
					}
					cgroup := newCgroupV2(id, r.cgroupRoot, relPath, r.cgroupControllers, r.pidMapper)
					res[id] = cgroup
					r.inodeMapper[inode] = cgroup
					if err != nil {
						return err
					}
				}

				return err
			}

			return nil
		},
	})

	return res, err
}

// inodeForFile returns the inode of the file at the given path and follows symlinks.
func inodeForFile(path string) (uint64, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return stat.Sys().(*syscall.Stat_t).Ino, nil
}

func readCgroupControllers(cgroupRoot string) (map[string]struct{}, error) {
	controllersMap := make(map[string]struct{})
	err := parseFile(defaultFileReader, path.Join(cgroupRoot, controllersFile), func(s string) error {
		controllers := strings.Fields(s)
		for _, c := range controllers {
			controllersMap[c] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(controllersMap) == 0 {
		return nil, fmt.Errorf("no cgroup controllers activated at: %s", path.Join(cgroupRoot, controllersFile))
	}

	return controllersMap, nil
}
