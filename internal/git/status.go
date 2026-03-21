package git

type Status struct {
	Files []FileStatus
}

type FileStatus struct {
	IndexStatus    byte
	WorkTreeStatus byte
	Path           string
}

func (s *Status) HasChanges() bool {
	return len(s.Files) > 0
}

func (s *Status) StagedCount() int {
	count := 0
	for _, f := range s.Files {
		if f.IndexStatus != ' ' && f.IndexStatus != '?' {
			count++
		}
	}
	return count
}

func (s *Status) UntrackedCount() int {
	count := 0
	for _, f := range s.Files {
		if f.IndexStatus == '?' {
			count++
		}
	}
	return count
}
