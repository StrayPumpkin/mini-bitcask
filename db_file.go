package minibitcask

import (
	"os"
	"path/filepath"
	"sync"
)

const FileName = "minibitcask.data"            //主数据文件夹
const MergeFileName = "minibitcask.data.merge" //合并时的临时文件

// DBFile 数据文件定义
type DBFile struct {
	File          *os.File   //文件句柄
	Offset        int64      //文件尾指针（下一个写入位置）
	HeaderBufPool *sync.Pool //头部缓冲池，划出一块可复用的内存空间，让多个读写操作共享使用。
}

func newInternal(fileName string) (*DBFile, error) {
	//打开或创建文件
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(fileName)
	if err != nil {
		return nil, err
	}
	// 创建一个 sync.Pool 结构体，并返回其指针
	pool := &sync.Pool{
		// New 字段是一个函数类型，当池中没有可用对象时，会调用这个函数创建新对象
		New: func() interface{} {
			return make([]byte, entryHeaderSize)
		},
	}
	return &DBFile{Offset: stat.Size(), File: file, HeaderBufPool: pool}, nil
}

// NewDBFile 创建一个新的数据文件
func NewDBFile(path string) (*DBFile, error) {
	//根据操作系统自动使用正确的路径分隔符。
	fileName := filepath.Join(path, FileName)
	return newInternal(fileName)
}

// NewMergeDBFile 新建一个合并时的数据文件
func NewMergeDBFile(path string) (*DBFile, error) {
	fileName := filepath.Join(path, MergeFileName)
	return newInternal(fileName)
}

// Read 从 offset 处开始读取
func (df *DBFile) Read(offset int64) (e *Entry, err error) {
	//借用缓冲区并归还
	buf := df.HeaderBufPool.Get().([]byte)
	defer df.HeaderBufPool.Put(buf)
	
	if _, err = df.File.ReadAt(buf, offset); err != nil {
		return
	}
	if e, err = Decode(buf); err != nil {
		return
	}

	offset += entryHeaderSize
	if e.KeySize > 0 {
		key := make([]byte, e.KeySize)
		if _, err = df.File.ReadAt(key, offset); err != nil {
			return
		}
		e.Key = key
	}

	offset += int64(e.KeySize)
	if e.ValueSize > 0 {
		value := make([]byte, e.ValueSize)
		if _, err = df.File.ReadAt(value, offset); err != nil {
			return
		}
		e.Value = value
	}
	return
}

// Write 写入 Entry
func (df *DBFile) Write(e *Entry) (err error) {
	enc, err := e.Encode()
	if err != nil {
		return err
	}
	_, err = df.File.WriteAt(enc, df.Offset)
	df.Offset += e.GetSize()
	return
}
