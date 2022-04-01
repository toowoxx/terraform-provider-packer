package provider

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

type State struct {
	Builds      map[string]*Build
	BuildsMutex sync.Mutex
}

type Build struct {
	ImageResourceData *resourceImageType
	Started           bool
	Finished          bool
	PreStartChan      chan interface{}
	FinishChan        chan bool
}

var state = &State{}

func init() {
	state.Builds = map[string]*Build{}
}

func (b *Build) Init() {
	b.FinishChan = make(chan bool)
	b.PreStartChan = make(chan interface{})
}

func (b *Build) AwaitCompletion() (success bool) {
	b.AwaitStart()
	return b.Finished || <-b.FinishChan
}

func (b *Build) AwaitStart() {
	<-b.PreStartChan
}

func (s *State) StartBuild(data *resourceImageType) error {
	if data.Name.Null || data.Name.Unknown {
		return nil
	}
	buildName := data.Name.Value
	if len(buildName) == 0 {
		return errors.New("build name must not be empty!")
	}

	s.BuildsMutex.Lock()
	defer s.BuildsMutex.Unlock()

	build, exists := s.Builds[buildName]
	if exists && build.Started {
		return fmt.Errorf("build name \"%s\" already exists, please specify a different one", buildName)
	} else if !exists {
		build = &Build{
			ImageResourceData: data,
		}
		build.Init()
		s.Builds[buildName] = build
	} else {
		build.ImageResourceData = data
	}

	build.Started = true
	build.PreStartChan <- nil
	close(build.PreStartChan)

	return nil
}

func (s *State) CompleteBuild(name string, success bool) {
	if len(name) == 0 {
		return
	}
	s.BuildsMutex.Lock()
	defer s.BuildsMutex.Unlock()

	build := s.Builds[name]
	build.FinishChan <- success
	build.Finished = true
	close(build.FinishChan)
}

func (s *State) preCreateBuild(name string) *Build {
	build := &Build{}
	build.Init()

	s.Builds[name] = build

	return build
}

func (s *State) GetBuild(name string) *Build {
	if len(name) == 0 {
		return nil
	}
	s.BuildsMutex.Lock()
	defer s.BuildsMutex.Unlock()

	build, exists := s.Builds[name]
	if !exists {
		build = s.preCreateBuild(name)
		s.Builds[name] = build
	}

	return build
}

func (s *State) RefreshBuildData(data *resourceImageType) {
	if data.Name.Null || data.Name.Unknown {
		return
	}
	buildName := data.Name.Value
	if len(buildName) == 0 {
		return
	}

	s.BuildsMutex.Lock()
	defer s.BuildsMutex.Unlock()

	build, exists := s.Builds[buildName]
	if exists && build.Started {
		return
	} else if !exists {
		build = &Build{
			ImageResourceData: data,
		}
		build.Init()
		s.Builds[buildName] = build
	}

	build.ImageResourceData = data

	if !build.Started {
		build.Started = true
		build.PreStartChan <- nil
		close(build.PreStartChan)
	}

	return
}
