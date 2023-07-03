package dhcpd

const bitsPerWord = 64

// bitSet is a sparse bitSet.  A nil *bitSet is an empty bitSet.
type bitSet struct {
	words map[uint64]uint64
}

// newBitSet returns a new bitset.
func newBitSet() (s *bitSet) {
	return &bitSet{
		words: map[uint64]uint64{},
	}
}

// isSet returns true if the bit n is set.
func (s *bitSet) isSet(n uint64) (ok bool) {
	if s == nil {
		return false
	}

	wordIdx := n / bitsPerWord
	bitIdx := n % bitsPerWord

	var word uint64
	word, ok = s.words[wordIdx]

	return ok && word&(1<<bitIdx) != 0
}

// set sets or unsets a bit.
func (s *bitSet) set(n uint64, ok bool) {
	if s == nil {
		return
	}

	wordIdx := n / bitsPerWord
	bitIdx := n % bitsPerWord

	word := s.words[wordIdx]
	if ok {
		word |= 1 << bitIdx
	} else {
		word &^= 1 << bitIdx
	}

	s.words[wordIdx] = word
}
