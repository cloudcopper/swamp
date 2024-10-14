TODO
====
- retention
-- starts 30 minutes after app startup
-- artifacts should be first marked as exipred (Artifact.State field)
-- expired artifacts deleted on second round

- artifacts checks functionality
-- starts 30 minutes after app startup
-- periodic with arbitrary interval or evenly distributed (should it be per project defined?) ArtifactsCheckInterval and ArtifactsCheckDuration - only one shall be defined
-- mark artifact broken (Artifact.State field)
-- if repo has no Repo.Broken defined then artifacts presents all the time
-- broken artifacts suppose to be moved on second cicle
-- broken as /dev/null means to be deleted
-- Handle manual artifact removal from artifact storage
-- reduce Repo.Size on artifact removal

- meta for artifacts. use similar to the checksum algo way???
-- with blacklist for some variables - i.e *PASSWORD* value should be *************
-- with url support (to have i.e. back links)

- artifacts download as separate file or aggregate (.zip)
-- broken artifacts can not be downloaded within single click (figure out how to make it complex for automation - i.e. present link with some random value instead of normal artifact id)
-- webui marks broken files in artifact as well whole artifact

CUSTOM.md - how to customize

- nicer ui
-- main page - shameless swamp promo (some backlink to gh and latest release?), instance summary, repos in short, and latest artifacts in short. Has way to download latest artifact from repo view and from artifact view.
-- repo page - repo summary and meta, repo artifacts summary, artifacts (pagination? calendar separation? meta search?)
-- artifact page - artifact details, files and their status, metas, meta filters
-- broken artifact warning page/mechanism ? should prohibit from simplest curl/wget downloads - sort of requires random URL instead valid file name?
- more nicer ui
-- meta values colorization humanization (i.e. automatically shows as symlink https:// and etc)
-- custom 404 ?

- tests - increate test coverage

- access log
- input web (the way to put over http new artifacts)
- abstract out storage
-- currently it is filesystem but may it be more flexible? minio?

- gorm -> goent ???