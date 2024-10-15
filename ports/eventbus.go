package ports

type Topic = string
type Event = []string
type EventBus interface {
	Shutdown()
	Pub(Topic, Event)
	Sub(Topic) chan Event
	Unsub(chan Event)
}

const (
	TopicRepoUpdated          Topic = "repo-updated"
	TopicArtifactUpdated      Topic = "artifact-updated"
	TopicInputUpdated         Topic = "input-updated"
	TopicInputFileModified    Topic = "input-file-modified"
	TopicDanglingRepoArtifact Topic = "dangling-repo-artifact"
	TopicBrokenRepoArtifact   Topic = "broken-repo-artifact"
)
