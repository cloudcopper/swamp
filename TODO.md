TODO
====
- nicer ui
-- main page - shameless swamp promo (some backlink to gh and latest release?), instance summary, repos in short, and latest artifacts in short. Has way to download latest artifact from repo view and from artifact view.
-- error pages

- broken artifacts shall not be possible to download within single click/direct url
- broken files shall not be possible to download within single click/direct url
-- figure out how to make it complex for automation - i.e. present link with some random value instead of normal artifact id

- layered fs with afero instead of custom?

- better configuration (atm params dropped here and there and it wont works well with proper DI)

CUSTOM.md - how to customize

- nicer ui
-- main page - artifacts pagination? 
-- repo page - artifacts pagination? calendar separation?
-- about page ?

- tests - increase test coverage

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

- meta filter at the page
- meta search

- custom renderer for more idiomatic layout support in html/template ?