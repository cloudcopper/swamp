TODO
====
- better configuration (atm params dropped here and there and it wont works well with proper DI)
- CUSTOM.md - how to customize
- Correct swamp-intro and readme

- broken artifacts shall not be possible to download within single click/direct url
- broken files shall not be possible to download within single click/direct url
-- figure out how to make it complex for automation - i.e. present link with some random value instead of normal artifact id

- layered fs with afero instead of custom?

- models.Repo.AritfactsCount (needed of we would like to have pagination working faster)
- front page artifacts limit(1) to speed up

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

- gorm -> goent ???
- uber fx or google wire ???

- archetypes for different artifacts/repos???

- meta filter at the page
- meta search
