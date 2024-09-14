Swamp - trivial artifactory
===========================

![](./20231122_100028.jpg)

TODO
====
- meta ??? remove Repo.Meta field and use similar to the checksum algo way
-- with blacklist for some variables?
-- meta is many to many?

- rescan artifacts at startup ?
- nicer ui (use template and repository to fetch Repo with Atrifacts info, probably also fillup info from fs too?)
- Use OptWithLogger to pass log into New... ?
- How to be with checksum algos?
- infra/config for repo_config? how to pass/embedd file and make nice vendoring? fs.embed layers?
- fsnotify to ports/adapters ?
- filesystem ports/adapters for tests and mocking? 
- test coverage
- tests!!!
- cut of infra out of adapters
- favicon.ico (to avoid 404 when browser goes in)
- input web (the way to put over http new artifacts)
