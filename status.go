package git

/*
#include <git2.h>
#include <git2/errors.h>

int _go_git_status_foreach(git_repository *repo, void *data);
*/
import "C"

import (
	"runtime"
	"unsafe"
)

type Status int

const (
	StatusCurrent         Status = C.GIT_STATUS_CURRENT
	StatusIndexNew               = C.GIT_STATUS_INDEX_NEW
	StatusIndexModified          = C.GIT_STATUS_INDEX_MODIFIED
	StatusIndexDeleted           = C.GIT_STATUS_INDEX_DELETED
	StatusIndexRenamed           = C.GIT_STATUS_INDEX_RENAMED
	StatusIndexTypeChange        = C.GIT_STATUS_INDEX_TYPECHANGE
	StatusWtNew                  = C.GIT_STATUS_WT_NEW
	StatusWtModified             = C.GIT_STATUS_WT_MODIFIED
	StatusWtDeleted              = C.GIT_STATUS_WT_DELETED
	StatusWtTypeChange           = C.GIT_STATUS_WT_TYPECHANGE
	StatusWtRenamed              = C.GIT_STATUS_WT_RENAMED
	StatusIgnored                = C.GIT_STATUS_IGNORED
)

type StatusEntry struct {
	Status         Status
	HeadToIndex    DiffDelta
	IndexToWorkdir DiffDelta
}

func statusEntryFromC(statusEntry *C.git_status_entry) StatusEntry {
	return StatusEntry {
		Status:         Status(statusEntry.status),
		HeadToIndex:    diffDeltaFromC(statusEntry.head_to_index),
		IndexToWorkdir: diffDeltaFromC(statusEntry.index_to_workdir),
	}
}

type StatusList struct {
	ptr *C.git_status_list
}

func newStatusListFromC(ptr *C.git_status_list) *StatusList {
	if ptr == nil {
		return nil
	}

	statusList := &StatusList{
		ptr: ptr,
	}

	runtime.SetFinalizer(statusList, (*StatusList).Free)
	return statusList
}

func (statusList *StatusList) Free() error {
	if statusList.ptr == nil {
		return ErrInvalid
	}
	runtime.SetFinalizer(statusList, nil)
	C.git_status_list_free(statusList.ptr)
	statusList.ptr = nil
	return nil
}

func (statusList *StatusList) ByIndex(index int) (StatusEntry, error) {
	if statusList.ptr == nil {
		return StatusEntry{}, ErrInvalid
	}
	ptr := C.git_status_byindex(statusList.ptr, C.size_t(index))
	return statusEntryFromC(ptr), nil
}

func (statusList *StatusList) EntryCount() (int, error) {
	if statusList.ptr == nil {
		return -1, ErrInvalid
	}
	return int(C.git_status_list_entrycount(statusList.ptr)), nil
}

const (
	StatusOptIncludeUntracked             = C.GIT_STATUS_OPT_INCLUDE_UNTRACKED
	StatusOptIncludeIgnored               = C.GIT_STATUS_OPT_INCLUDE_IGNORED
	StatusOptIncludeUnmodified            = C.GIT_STATUS_OPT_INCLUDE_UNMODIFIED
	StatusOptExcludeSubmodules            = C.GIT_STATUS_OPT_EXCLUDE_SUBMODULES
	StatusOptRecurseUntrackedDirs         = C.GIT_STATUS_OPT_RECURSE_UNTRACKED_DIRS
	StatusOptDisablePathspecMatch         = C.GIT_STATUS_OPT_DISABLE_PATHSPEC_MATCH
	StatusOptRecurseIgnoredDirs           = C.GIT_STATUS_OPT_RECURSE_IGNORED_DIRS
	StatusOptRenamesHeadToIndex           = C.GIT_STATUS_OPT_RENAMES_HEAD_TO_INDEX
	StatusOptRenamesIndexToWorkdir        = C.GIT_STATUS_OPT_RENAMES_INDEX_TO_WORKDIR
	StatusOptSortCaseSensitively          = C.GIT_STATUS_OPT_SORT_CASE_SENSITIVELY
	StatusOptSortCaseInsensitively        = C.GIT_STATUS_OPT_SORT_CASE_INSENSITIVELY
	StatusOptRenamesFromRewrites          = C.GIT_STATUS_OPT_RENAMES_FROM_REWRITES
	StatusOptNoRefresh                    = C.GIT_STATUS_OPT_NO_REFRESH
	StatusOptUpdateIndex                  = C.GIT_STATUS_OPT_UPDATE_INDEX
)

type StatusShow int

const (
	StatusShowIndexAndWorkdir StatusShow = C.GIT_STATUS_SHOW_INDEX_AND_WORKDIR
	StatusShowIndexOnly                  = C.GIT_STATUS_SHOW_INDEX_ONLY
	StatusShowWorkdirOnly                = C.GIT_STATUS_SHOW_WORKDIR_ONLY
)

type StatusOptions struct {
	Version  int
	Show     StatusShow
	Flags    int
	Pathspec []string
}

func (opts *StatusOptions) toC() *C.git_status_options {
	if opts == nil {
		return nil
	}

	cpathspec := C.git_strarray{}
	if opts.Pathspec != nil {
		cpathspec.count = C.size_t(len(opts.Pathspec))
		cpathspec.strings = makeCStringsFromStrings(opts.Pathspec)
		defer freeStrarray(&cpathspec)
	}

	copts := &C.git_status_options{
		version:  C.GIT_STATUS_OPTIONS_VERSION,
		show:     C.git_status_show_t(opts.Show),
		flags:    C.uint(opts.Flags),
		pathspec: cpathspec,
	}

	return copts
}

func (v *Repository) StatusList(opts *StatusOptions) (*StatusList, error) {
	var ptr *C.git_status_list
	var copts *C.git_status_options

	if opts != nil {
		copts = opts.toC()
	} else {
		copts = &C.git_status_options{}
		ret := C.git_status_init_options(copts, C.GIT_STATUS_OPTIONS_VERSION)
		if ret < 0 {
			return nil, MakeGitError(ret)
		}
	}

	ret := C.git_status_list_new(&ptr, v.ptr, copts)
	if ret < 0 {
		return nil, MakeGitError(ret)
	}
	return newStatusListFromC(ptr), nil
}


func (v *Repository) StatusFile(path string) (Status, error) {
	var statusFlags C.uint
	cPath := C.CString(path)
	ret := C.git_status_file(&statusFlags, v.ptr, cPath)
	if ret < 0 {
		return 0, MakeGitError(ret)
	}
	return Status(statusFlags), nil
}

type StatusCallback func(path string, status Status) int

//export fileStatusForeach
func fileStatusForeach(_path *C.char, _flags C.uint, _payload unsafe.Pointer) C.int {
	path := C.GoString(_path)
	flags := Status(_flags)

	cb := (*StatusCallback)(_payload)
	return C.int((*cb)(path, flags))
}

func (v *Repository) StatusForeach(callback StatusCallback) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	ret := C._go_git_status_foreach(v.ptr, unsafe.Pointer(&callback))

	if ret < 0 {
		return MakeGitError(ret)
	}
	return nil
}
