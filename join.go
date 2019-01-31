package aorm

type JoinType uint

func (ipo JoinType) IsInner() bool {
	return (ipo & JoinInner) != 0
}

func (ipo JoinType) IsLeft() bool {
	return ipo == 0 || (ipo&JoinLeft) != 0
}

func (ipo JoinType) IsRight() bool {
	return ipo == 0 || (ipo&JoinRight) != 0
}

func (ipo JoinType) String() string {
	if ipo.IsLeft() {
		return "LEFT"
	}
	if ipo.IsRight() {
		return "RIGHT"
	}
	if ipo.IsInner() {
		return "INNER"
	}
	return ""
}

const (
	JoinLeft JoinType = iota
	JoinInner
	JoinRight
)
