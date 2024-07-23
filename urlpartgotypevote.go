package protocgenghe

type URLPartGoTypeVote struct {
	CaptureIndex int
	Votes        map[string]int
	UnknownTypes []string
}

func NewURLPartGoTypeVote(captureIndex int) *URLPartGoTypeVote {
	return &URLPartGoTypeVote{
		CaptureIndex: captureIndex,
		Votes:        make(map[string]int),
	}
}

func (vote *URLPartGoTypeVote) Vote(goType string) {
	if _, ok := vote.Votes[goType]; ok {
		vote.Votes[goType]++
	} else {
		vote.Votes[goType] = 1
		if _, supportType := DefaultURLPartTypePatterns[goType]; !supportType {
			vote.UnknownTypes = append(vote.UnknownTypes, goType)
		}
	}
}
