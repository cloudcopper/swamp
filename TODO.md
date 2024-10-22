TODO
====
- broken artifacts shall not be possible to download within single click/direct url
- broken files shall not be possible to download within single click/direct url
-- figure out how to make it complex for automation - i.e. present link with some random value instead of normal artifact id

- layered fs with afero instead of custom?

- better configuration (atm params dropped here and there and it wont works well with proper DI)

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

- handle manual artifact removal from artifact storage
- handle manual artifact adding to artifact storage

- access log
- input web (the way to put over http new artifacts)
- abstract out storage
-- currently it is filesystem but may it be more flexible? minio?

- blacklist for meta keys
- blacklist for meta values - i.e *PASSWORD* should be *************

- gorm -> goent ???
- uber fx or google wire ???

- archetypes for different artifacts/repos???
