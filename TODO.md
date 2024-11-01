TODO
====
- CUSTOM.md - how to customize
- Correct swamp-intro and readme
- Release v0.1.0

- broken artifacts shall not be possible to download within single click/direct url
- broken files shall not be possible to download within single click/direct url
-- figure out how to make it complex for automation - i.e. present link with some random value instead of normal artifact id

- layered fs with afero instead of custom?

- nicer ui
-- main page - artifacts pagination? 
-- repo page - artifacts pagination? calendar separation?
-- about page ?

- tests - increase test coverage
- better configuration (atm params are at package level and it might not works well with massive testing or proper DI)

- handle manual artifact removal from artifact storage
- handle manual artifact adding to artifact storage

- access log
- input web (the way to put over http new artifacts)
- abstract out storage
-- currently it is filesystem but may it be more flexible? minio?

- gorm -> goent ???
- uber fx or google wire ???

- archetypes for different artifacts/repos???

- meta filter at the page
- meta search
