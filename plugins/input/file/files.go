package file

import (
	//	"encoding/json"
	"github.com/luopengift/golibs/file"
	"github.com/luopengift/golibs/logger"
	"github.com/luopengift/transport/pipeline"
)

type FilesInput struct {
	Files []*file.Tail
    buf chan []byte
}

func NewFilesInput() *FilesInput {
	return new(FilesInput)
}

type FilesConfig struct {
	Path    []string `json:"path"`
	EndStop bool     `json:"endstop"`
}

func (in *FilesInput) Init(config pipeline.Configer) error {
	cfg := FilesConfig{}
	err := config.Parse(&cfg)
	if err != nil {
		logger.Error("parse error:%v", err)
		return err
	}
    for _,path := range cfg.Path {
	    tail := file.NewTail(path, file.TimeRule)
	    if cfg.EndStop {
		    tail.EndStop(true)
	    }
        in.Files = append(in.Files, tail)
    }
    in.buf = make(chan []byte, 1000)
	return nil
}

func (in *FilesInput) Read(p []byte) (int, error) {
	n := copy(p, <-in.buf)
    return n, nil
}

func (in *FilesInput) Start() error {
    for _, tail := range in.Files {
	    go func(t *file.Tail) {
            t.ReadLine()
            for msg := range t.NextLine() {
                in.buf <- msg
            }
        }(tail)
    }

	return nil
}

func (in *FilesInput) Close() error {
    for _, tail := range in.Files {
        tail.Close()
    }
    return nil
}

func init() {
	pipeline.RegistInputer("files", NewFilesInput())
}
