package other

import (
	"io"

	ssz "github.com/ferranbt/fastssz"
)

type Case4Interface struct {
}

func (c *Case4Interface) SizeSSZ() (size int) {
	return 96
}

func (s *Case4Interface) MarshalSSZTo(buf []byte) ([]byte, error) {
	return nil, nil
}

func (s *Case4Interface) HashTreeRootWith(hh ssz.HashWalker) (err error) {
	return
}

func (s *Case4Interface) UnmarshalSSZ(buf []byte) error {
	return nil
}

func (s *Case4Interface) UnmarshalSSZTail(buf []byte) ([]byte, error) {
	return nil, nil
}

func (s *Case4Interface) Decode(src io.Reader, limit int) (int, error) {
	return 0, nil
}

func (s *Case4Interface) Encode(dst io.Writer) (int, error) {
	return 0, nil
}

type Case4FixedSignature [96]byte

type Case4Bytes []byte
