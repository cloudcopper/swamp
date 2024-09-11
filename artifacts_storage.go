package main

type ArtifactID string

type ArtifactsStorage interface {
	NewArtifacts(*Repo, []string, ArtifactID) (ArtifactID, error)
}
