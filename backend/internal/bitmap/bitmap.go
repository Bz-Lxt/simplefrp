package bitmap

func New(bits int) *Set {
	if bits <= 0 {
		bits = 64
	}
	n := (bits + 63) / 64
	return &Set{bits: bits, words: make([]uint64, n)}
}

type Set struct {
	bits  int
	words []uint64
}

func (s *Set) Cap() int { return s.bits }

func (s *Set) Set(i int) {
	if i < 0 || i >= s.bits {
		return
	}
	s.words[i/64] |= 1 << uint(i%64)
}

func (s *Set) Clear(i int) {
	if i < 0 || i >= s.bits {
		return
	}
	s.words[i/64] &^= 1 << uint(i%64)
}

func (s *Set) Has(i int) bool {
	if i < 0 || i >= s.bits {
		return false
	}
	return s.words[i/64]&(1<<uint(i%64)) != 0
}

func (s *Set) Toggle(i int) {
	if i < 0 || i >= s.bits {
		return
	}
	s.words[i/64] ^= 1 << uint(i%64)
}

func (s *Set) Count() int {
	n := 0
	for _, w := range s.words {
		n += popcount(w)
	}
	return n
}

func (s *Set) NextClear(from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i < s.bits; i++ {
		if !s.Has(i) {
			return i
		}
	}
	return -1
}

func (s *Set) NextSet(from int) int {
	if from < 0 {
		from = 0
	}
	for i := from; i < s.bits; i++ {
		if s.Has(i) {
			return i
		}
	}
	return -1
}

func (s *Set) Or(other *Set) {
	if other == nil {
		return
	}
	n := len(s.words)
	if len(other.words) < n {
		n = len(other.words)
	}
	for i := 0; i < n; i++ {
		s.words[i] |= other.words[i]
	}
}

func (s *Set) And(other *Set) {
	if other == nil {
		return
	}
	n := len(s.words)
	if len(other.words) < n {
		n = len(other.words)
	}
	for i := 0; i < n; i++ {
		s.words[i] &= other.words[i]
	}
	for i := n; i < len(s.words); i++ {
		s.words[i] = 0
	}
}

func (s *Set) Clone() *Set {
	out := New(s.bits)
	copy(out.words, s.words)
	return out
}

func (s *Set) Reset() {
	for i := range s.words {
		s.words[i] = 0
	}
}

func popcount(x uint64) int {
	n := 0
	for x != 0 {
		x &= x - 1
		n++
	}
	return n
}
