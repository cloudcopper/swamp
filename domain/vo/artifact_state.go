package vo

type ArtifactState int

const (
	ArtifactIsOK      ArtifactState = 0
	ArtifactIsBroken  ArtifactState = 1
	ArtifactIsExpired ArtifactState = 2
)

func (s ArtifactState) IsOK() bool {
	return s == ArtifactIsOK
}

func (s ArtifactState) IsBroken() bool {
	return s&ArtifactIsBroken != 0
}

func (s ArtifactState) IsExpired() bool {
	return s&ArtifactIsExpired != 0
}
