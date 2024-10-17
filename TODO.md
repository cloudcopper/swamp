TODO
====
- meta for artifacts
-- with blacklist for some variables - i.e *PASSWORD* value should be *************
-- remove model artifact should remove its metas too

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

- artifacts checks functionality
-- Mark broken files in file.State
-- Handle manual artifact removal from artifact storage

- access log
- input web (the way to put over http new artifacts)
- abstract out storage
-- currently it is filesystem but may it be more flexible? minio?

- gorm -> goent ???
- uber fx ???