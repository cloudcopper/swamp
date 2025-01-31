package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cloudcopper/random"
	"github.com/cloudcopper/swamp/adapters/http"
	"github.com/cloudcopper/swamp/adapters/http/controllers"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/infra/config"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
	"github.com/cloudcopper/swamp/ports"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/afero"
)

var (
	numRepos         = []int{1, 20}
	numRepoName      = []int{2, 5}
	numDescSentences = []int{1, 5}
	numRepoIdLetters = []int{3, 5}
	numRepoIdNumbers = []int{0, 3}
	retentions       = []types.Duration{
		0,
		types.Duration(30 * time.Minute),
		types.Duration(1 * time.Hour),
		types.Duration(24 * time.Hour),
		types.Duration(36 * time.Hour),
		types.Duration(7 * 24 * time.Hour),
		types.Duration(30 * 24 * time.Hour),
		types.Duration(3 * 30 * 24 * time.Hour),
		types.Duration(365 * 24 * time.Hour),
	}
	dirs = func() []string {
		root := "/"
		a := []string{}

		filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				return nil
			}
			a = append(a, path)
			if len(a) > 200 {
				return fmt.Errorf("done")
			}
			return nil
		})

		return a
	}()
	brokens = []string{
		"",
		"/dev/null",
		rs(dirs),
	}
	numRepoMetas          = []int{0, 10}
	numRepoMetaNames      = []int{1, 4}
	numRepoMetaValueTexts = []int{1, 4}

	numArtifacts = []int{30, 600}
	numFiles     = []int{1, 30}
)

var (
	rv = random.Value
	rw = random.Words
	rs = random.Element[string]
	ri = random.Element[int]
)

func main() {
	log := slog.Default()
	err := app(log)
	_ = err
}

// Massive copy paste from app.go
func app(log *slog.Logger) error {
	// Force development environment
	os.Setenv("GO_ENV", "development")

	// Create layered filesystem
	fs, err := infra.NewLayerFileSystem(os.Getwd)
	if err != nil {
		log.Error("unable to create layered filesystem!!!", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetLayerFilesystemError)
	}

	// Open database
	driver := infra.DriverSqlite
	source := infra.SourceSqliteInMemory
	db, closeDb, err := infra.NewDatabase(log, driver, source)
	if err != nil {
		log.Error("unable to create database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, errors.RetCreateDatabaseError)
	}
	defer closeDb()
	// Sync database
	if err := db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta), new(models.ArtifactFile)); err != nil {
		log.Error("unable sync database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, errors.RetMigrateDatabaseError)
	}
	// Create repositories
	realFS := afero.NewOsFs()
	var repoRepository domain.RepoRepository
	repoRepository, err = repository.NewRepoRepository(db, realFS)
	if err != nil {
		log.Error("unable create repo repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateRepoRepositoryError)
	}
	repoRepository = NewBadRepoRepository(repoRepository)
	var artifactRepository domain.ArtifactRepository
	artifactRepository, err = repository.NewArtifactRepository(db, realFS)
	if err != nil {
		log.Error("unable create artifact repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateArtifactRepositoryError)
	}
	artifactRepository = NewBadArtifactRepository(artifactRepository)
	repositories := repository.NewRepositories(repoRepository, artifactRepository)

	// Perform neccesery startup operations
	if err := startup(log, repositories); err != nil {
		return err
	}

	// Create router
	router := http.NewRouter(log)
	// Create render object
	// It also loads templates
	render := infra.NewRender(fs, "layout")
	// Create controllers
	frontPageController := controllers.NewFrontPageController(log, render, repositories)
	repoContoller := controllers.NewRepoController(log, render, repoRepository)
	artifactController := controllers.NewArtifactController(log, render, artifactRepository, fakeStorage)
	aboutPageController := controllers.NewAboutPageController(log, render)
	// Add routes
	router.Get("/", frontPageController.Index)
	router.Get("/about", aboutPageController.Index)
	router.Get("/repo/{repoID}/artifact/{artifactID}/file/*", artifactController.DownloadSingleFile)
	router.Get("/repo/{repoID}/artifact/{artifactID}.tar.gz", artifactController.DownloadGzip)
	router.Get("/repo/{repoID}/artifact/{artifactID}.zip", artifactController.DownloadZip)
	router.Get("/repo/{repoID}/artifact/{artifactID}", artifactController.Get)
	router.Get("/repo/{repoID}", repoContoller.Get)
	// Static file handler
	fileServer := http.FileServer(http.FS(fs))
	router.Handle("/static/*", fileServer)
	// 404 handler
	router.NotFound(frontPageController.NotFound)
	// Create http server
	// The router must has all routes already
	// It will start server in separate goroutine
	addr := config.Listen
	httpServer, err := infra.NewWebServer(log, addr, router)
	if err != nil {
		log.Error("unable create web server", slog.Any("err", err), slog.String("addr", addr))
		return lib.NewErrorCode(err, errors.RetCreateWebServerError)
	}

	// Add ctrl-c shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	log.Info("press ctrl-c to exit")
	// Wait for ctrl-c
	<-c

	// Close http server
	httpServer.Close()
	return nil
}

// Prefill database with random data
func startup(log ports.Logger, repos domain.Repositories) error {
	maxRepos := rv(numRepos)
	log.Info("create repos", slog.Any("maxRepos", maxRepos))
	for n := 0; n < maxRepos; n++ {
		name := rw(numRepoName)
		repoID := genRepoID(name, numRepoIdLetters, numRepoIdNumbers)

		meta := models.RepoMetas{}
		for a := 0; a < rv(numRepoMetas); a++ {
			name, value := randomMeta()
			meta = append(meta, &models.RepoMeta{
				RepoID: repoID,
				Key:    name,
				Value:  value,
			})
		}

		repo := &models.Repo{
			RepoID:      repoID,
			Name:        name,
			Description: random.Sentences(numDescSentences),
			Input:       rs(dirs),
			Storage:     rs(dirs),
			Retention:   random.Element(retentions),
			Broken:      rs(append(brokens, dirs...)),
			Size:        0,
			Meta:        meta,
		}

		err := repos.Repo().Create(repo)
		if err != nil {
			log.Error("unable create repo", slog.Any("err", err))
			continue
		}
		log.Info("created repo", slog.Any("repoID", repoID))

		for m := 0; m < rv(numArtifacts); m++ {
			artifactID := genArtifactID()

			meta := []*models.ArtifactMeta{}
			for x := 0; x < rv([]int{5, 100}); x++ {
				name, value := randomMeta()
				m := &models.ArtifactMeta{
					Key:   strings.ToUpper(strings.ReplaceAll(name, " ", "_")),
					Value: value,
				}
				meta = append(meta, m)
			}
			// add secret/password simulation entry to be able test blacklisting
			meta = append(meta, []*models.ArtifactMeta{
				{
					Key:   strings.ToUpper(random.Word()) + "_PASSWORD", // The 'PASSWORD' shall blacklist value
					Value: random.Word(),
				},
				{
					Key:   "SECRET_" + strings.ToUpper(random.Word()), // The 'SECRET' shall blacklist value
					Value: random.Words([]int{2, 4}),
				}, {
					Key:   "_" + strings.ToUpper(strings.ReplaceAll(name, " ", "_")), // The ^_ shall blacklist key/value
					Value: random.Words([]int{2, 4}),
				}}...)

			createdAt := int64(rv([]int{0, int(time.Now().UTC().Unix())}))
			expiredAt := createdAt + int64(repo.Retention/1000000000)
			state := vo.ArtifactState(ri([]int{0, 0, 0, 0, 0}))
			if expiredAt != createdAt && expiredAt < time.Now().UTC().Unix() {
				state |= vo.ArtifactIsExpired
			} else {
				state &= ^vo.ArtifactIsExpired
			}
			checksum := random.Sha256()
			files := getArtifactFiles(checksum, createdAt)
			size := 0
			for _, f := range files {
				size += int(f.Size)
			}
			artifact := &models.Artifact{
				RepoID:     repoID,
				ArtifactID: artifactID,
				Storage:    repo.Storage,
				Size:       types.Size(size),
				State:      state,
				CreatedAt:  createdAt,
				ExpiredAt:  expiredAt,
				Checksum:   checksum,
				Meta:       meta,
				Files:      files,
			}
			err := repos.Artifact().Create(artifact)
			if err != nil {
				log.Error("unable create artifact", slog.Any("err", err))
				continue
			}
			log.Info("created artifact", slog.Any("repoID", repoID), slog.Any("artifactID", artifactID))
		}
	}

	return nil
}

func genRepoID(name string, l, n []int) string {
	repoID := strings.ReplaceAll(name, " ", "_")
	repoID = repoID[:rv(l)]
	repoID = strings.ToLower(repoID)
	if x := rv(n); x > 0 {
		repoID += "-"
		for n := 0; n < x; n++ {
			digit := "0123456789"
			repoID += string(digit[rv([]int{0, 9})])
		}
	}
	return repoID
}

func genArtifactID() string {
	switch rs([]string{"semver", "hash", "dirty", "short-hash", "uuid", "ulid"}) {
	case "hash":
		return random.Sha256()
	case "short-hash":
		return random.Sha256()[:8]
	case "dirty":
		return fmt.Sprintf("v%s-%d-%s-dirty", random.SemVer(), rv([]int{1, 10}), random.Sha256()[:8])
	case "uuid":
		return uuid.New().String()
	case "ulid":
		return ulid.Make().String()
	}
	return random.SemVer()
}

var countGeneratedFile = int(0)

func getArtifactFiles(checksum string, createdAt int64) models.ArtifactFiles {
	_ = createdAt
	// Generate artifacts
	files := models.ArtifactFiles{}
	for x := 0; x < rv(numFiles); x++ {
		file := &models.ArtifactFile{
			Name:  filepath.Join(random.Filepath(3), random.FileName(3)),
			Size:  types.Size(rv([]int{128, 150000000})),
			State: vo.ArtifactState(random.Element([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1})),
		}
		countGeneratedFile++
		if countGeneratedFile%(numFiles[1]/6) == 0 {
			// break file name
			// the FakeStorage.OpenFile will return error on file with 'bad'
			// so we can simulate cases when file is "broken"
			file.Name += "bad.txt"
		} else if countGeneratedFile%(numFiles[1]/5) == 0 {
			// break file name
			// the FakeStorage.OpenFile may return error on file with 'flacky'
			// so we can simulate cases when file is "broken" randomly
			file.Name += "flacky.txt"
		}
		files = append(files, file)
	}

	files = append(files,
		&models.ArtifactFile{ // Generate _export.txt
			Name:  "_export.txt",
			Size:  types.Size(rv([]int{1024, 4096})),
			State: vo.ArtifactIsOK,
		},
		&models.ArtifactFile{ // Generate _createdAt.txt
			Name:  "_createdAt.txt",
			Size:  types.Size(rv([]int{32, 64})),
			State: vo.ArtifactIsOK,
		},
		&models.ArtifactFile{ // Generate checksum file
			Name:  checksum + ".sha256sum",
			Size:  types.Size(rv([]int{128, 1024})),
			State: vo.ArtifactIsOK,
		})

	return files
}

func randomMeta() (key, value string) {
	tld := []string{"com", "org", "net"}
	name, value := rw(numRepoMetaNames), ""
	switch rs([]string{"text", "http://", "https://", "mailto:"}) {
	case "text":
		value = rw(numRepoMetaValueTexts)
		value = strings.ToUpper(value[:1]) + value[1:]
	case "http://":
		value = "http://" + strings.ReplaceAll(rw([]int{1, 3}), " ", ".") + "." + rs(tld) + "/" + strings.ReplaceAll(rw([]int{0, 4}), " ", "/")
	case "https://":
		value = "https://" + strings.ReplaceAll(rw([]int{1, 3}), " ", ".") + "." + rs(tld) + "/" + strings.ReplaceAll(rw([]int{0, 4}), " ", "/")
	case "mailto:":
		value = "mailto:" + strings.ReplaceAll(rw([]int{1, 3}), " ", ".") + "@" + strings.ReplaceAll(rw([]int{1, 2}), " ", ".") + "." + rs(tld)
	}
	return name, value
}

var fakeStorage = &FakeStorage{
	fs: afero.NewMemMapFs(),
}

type FakeStorage struct {
	fs ports.FS
	n  int
}

func (*FakeStorage) NewArtifact(ports.FS, string, []string, string, models.ArtifactID) (*ports.NewArtifactInfo, error) {
	panic("not expected to be called atm!!!")
}
func (*FakeStorage) RemoveArtifact(string, models.ArtifactID) error {
	panic("not expected to be called atm!!!")
}
func (s *FakeStorage) OpenFile(storage, artifactID, filename string) (ports.File, error) {
	if strings.Contains(artifactID, "bad") {
		return nil, lib.Error("fake error - artifactID contains 'bad' substring!!!")
	}
	if strings.Contains(filename, "bad") {
		return nil, lib.Error("fake error - filename contains 'bad' substring!!!")
	}
	if strings.Contains(filename, "flacky") {
		s.n++
		if s.n%3 == 0 {
			return nil, fmt.Errorf("fake error - flacky value reach %v", s.n)
		}
	}

	const name = "/tmp/dump.txt"
	afero.WriteFile(s.fs, name, []byte(random.Sentences([]int{4, 10})), 0o664)
	f, err := s.fs.Open(name)
	return f, err
}

type badRepoRepository struct {
	repo          domain.RepoRepository
	lastFindByID  string
	countFindByID int
}

func NewBadRepoRepository(repo domain.RepoRepository) domain.RepoRepository {
	return &badRepoRepository{
		repo: repo,
	}
}

func (b *badRepoRepository) Create(model *models.Repo) error {
	return b.repo.Create(model)
}
func (b *badRepoRepository) FindAll(flags ...interface{}) ([]*models.Repo, error) {
	b.lastFindByID = ""
	return b.repo.FindAll(flags...)
}
func (b *badRepoRepository) FindByID(id models.RepoID, flags ...interface{}) (*models.Repo, error) {
	if b.lastFindByID != id {
		b.lastFindByID = id
		b.countFindByID = 0
	}
	b.countFindByID++
	if b.countFindByID > 10 {
		return nil, fmt.Errorf("fake error while quering repo '%v'", id)
	}
	if b.countFindByID > 5 {
		return nil, ports.ErrRecordNotFound
	}

	return b.repo.FindByID(id, flags...)
}
func (b *badRepoRepository) IterateAll(callback func(*models.Repo) (bool, error)) error {
	return b.repo.IterateAll(callback)
}

type badArtifactRepository struct {
	repo          domain.ArtifactRepository
	lastFindByID  string
	countFindByID int
}

func NewBadArtifactRepository(repo domain.ArtifactRepository) domain.ArtifactRepository {
	return &badArtifactRepository{
		repo: repo,
	}
}

func (b *badArtifactRepository) Create(model *models.Artifact) error {
	return b.repo.Create(model)
}
func (b *badArtifactRepository) Update(model *models.Artifact) error {
	return b.repo.Update(model)
}
func (b *badArtifactRepository) Delete(model *models.Artifact) error {
	return b.repo.Delete(model)
}
func (b *badArtifactRepository) FindAll() ([]*models.Artifact, error) {
	b.lastFindByID = ""
	return b.repo.FindAll()
}
func (b *badArtifactRepository) FindAllTimeExpired(now int64) ([]*models.Artifact, error) {
	return b.repo.FindAllTimeExpired(now)
}
func (b *badArtifactRepository) FindAllStatusExpired(flags ...interface{}) ([]*models.Artifact, error) {
	return b.repo.FindAllStatusExpired(flags...)
}
func (b *badArtifactRepository) FindAllStatusNotBroken() ([]*models.Artifact, error) {
	return b.repo.FindAllStatusNotBroken()
}
func (b *badArtifactRepository) FindAllStatusBroken(flags ...interface{}) ([]*models.Artifact, error) {
	return b.repo.FindAllStatusBroken(flags...)
}
func (b *badArtifactRepository) FindByID(repoID models.RepoID, artifactID models.ArtifactID, flags ...interface{}) (*models.Artifact, error) {
	if b.lastFindByID != artifactID {
		b.lastFindByID = artifactID
		b.countFindByID = 0
	}
	b.countFindByID++
	if b.countFindByID > 10 {
		return nil, fmt.Errorf("fake error while quering repo '%v' artifact '%v'", repoID, artifactID)
	}
	if b.countFindByID > 5 {
		return nil, ports.ErrRecordNotFound
	}

	return b.repo.FindByID(repoID, artifactID, flags...)
}
func (b *badArtifactRepository) IterateAll(callback func(*models.Artifact) (bool, error)) error {
	return b.repo.IterateAll(callback)
}
