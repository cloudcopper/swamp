package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cloudcopper/swamp/adapters/http"
	"github.com/cloudcopper/swamp/adapters/http/controllers"
	"github.com/cloudcopper/swamp/adapters/repository"
	"github.com/cloudcopper/swamp/domain"
	"github.com/cloudcopper/swamp/domain/errors"
	"github.com/cloudcopper/swamp/domain/models"
	"github.com/cloudcopper/swamp/domain/vo"
	"github.com/cloudcopper/swamp/infra"
	"github.com/cloudcopper/swamp/lib"
	"github.com/cloudcopper/swamp/lib/types"
	"github.com/cloudcopper/swamp/ports"
	"github.com/go-loremipsum/loremipsum"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"golang.org/x/exp/rand"
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
	brokens = []string{
		"",
		"/dev/null",
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
	numRepoMetas          = []int{0, 10}
	numRepoMetaNames      = []int{1, 4}
	numRepoMetaValueTexts = []int{1, 4}

	numArtifacts = []int{30, 600}
	numFiles     = []int{1, 30}
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
	if err := db.AutoMigrate(new(models.Repo), new(models.RepoMeta), new(models.Artifact), new(models.ArtifactMeta)); err != nil {
		log.Error("unable sync database", slog.Any("err", err), slog.String("driver", driver), slog.String("source", source))
		return lib.NewErrorCode(err, errors.RetMigrateDatabaseError)
	}
	// Create repositories
	repoRepository, err := repository.NewRepoRepository(db)
	if err != nil {
		log.Error("unable create repo repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateRepoRepositoryError)
	}
	artifactRepository, err := repository.NewArtifactRepository(db)
	if err != nil {
		log.Error("unable create artifact repository", slog.Any("err", err))
		return lib.NewErrorCode(err, errors.RetCreateArtifactRepositoryError)
	}
	repositories := repository.NewRepositories(repoRepository, artifactRepository)

	// Perform neccesery startup operations
	if err := startup(log, repositories); err != nil {
		return err
	}

	// Create router
	router := http.NewRouter(log)
	// Create render object
	// It also loads templates
	render := infra.NewRender(fs)
	// Create controllers
	frontPageController := controllers.NewFrontPageController(log, render, repositories)
	repoContoller := controllers.NewRepoController(log, render, repoRepository)
	artifactController := controllers.NewArtifactController(log, render, artifactRepository, fakeStorage)
	// Add routes
	router.Get("/", frontPageController.Index)
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
	addr := ":8080"
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
	rand.Seed(uint64(time.Now().UnixNano()))

	maxRepos := random(numRepos)
	log.Info("create repos", slog.Any("maxRepos", maxRepos))
	for n := 0; n < maxRepos; n++ {
		name := genWords(numRepoName)
		repoID := genRepoID(name, numRepoIdLetters, numRepoIdNumbers)

		meta := models.RepoMetas{}
		for a := 0; a < random(numRepoMetas); a++ {
			tld := []string{"com", "org", "net"}
			name, value := genWords(numRepoMetaNames), ""
			switch R([]string{"text", "http://", "https://", "mailto:"}) {
			case "text":
				value = genWords(numRepoMetaValueTexts)
				value = strings.ToUpper(value[:1]) + value[1:]
			case "http://":
				value = "http://" + strings.ReplaceAll(genWords([]int{1, 3}), " ", ".") + "." + R(tld) + "/" + strings.ReplaceAll(genWords([]int{0, 4}), " ", "/")
			case "https://":
				value = "https://" + strings.ReplaceAll(genWords([]int{1, 3}), " ", ".") + "." + R(tld) + "/" + strings.ReplaceAll(genWords([]int{0, 4}), " ", "/")
			case "mailto:":
				value = "mailto:" + strings.ReplaceAll(genWords([]int{1, 3}), " ", ".") + "@" + strings.ReplaceAll(genWords([]int{1, 2}), " ", ".") + "." + R(tld)
			}
			meta = append(meta, &models.RepoMeta{
				RepoID: repoID,
				Key:    name,
				Value:  value,
			})
		}

		repo := &models.Repo{
			ID:          repoID,
			Name:        name,
			Description: gen.Sentences(random(numDescSentences)),
			Input:       R(dirs),
			Storage:     R(dirs),
			Retention:   R(retentions),
			Broken:      R(append(brokens, dirs...)),
			Size:        0,
			Meta:        meta,
		}

		err := repos.Repo().Create(repo)
		if err != nil {
			log.Error("unable create repo", slog.Any("err", err))
			continue
		}
		log.Info("created repo", slog.Any("repoID", repoID))

		for m := 0; m < random(numArtifacts); m++ {
			artifactID := genArtifactID()

			meta := []*models.ArtifactMeta{}
			for x := 0; x < random([]int{5, 100}); x++ {
				m := &models.ArtifactMeta{
					Key:   strings.ToUpper(strings.ReplaceAll(genWords([]int{1, 3}), " ", "_")),
					Value: genWords([]int{1, 5}),
				}
				meta = append(meta, m)
			}

			artifact := &models.Artifact{
				RepoID:    repoID,
				ID:        artifactID,
				Size:      types.Size(random([]int{1024, 150 * 1024 * 1024})),
				State:     vo.ArtifactState(random([]int{0, 3})),
				CreatedAt: int64(random([]int{0, int(time.Now().UTC().Unix())})),
				Checksum:  genChecksum(),
				Meta:      meta,
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

// The random returns value between a[0] and a[1]
func random(a []int) int {
	min, max := a[0], a[1]
	return rand.Intn(max-min+1) + min
}

// R returns random element of a
func R[T any](a []T) T {
	return a[random([]int{0, len(a) - 1})]
}

func genWords(r []int) string {
	name := ""
	for x := 0; x < random(r); x++ {
		if x != 0 {
			name += " "
		}
		name += gen.Word()
	}
	return name
}

func genRepoID(name string, l []int, n []int) string {
	repoID := strings.ReplaceAll(name, " ", "_")
	repoID = repoID[:random(l)]
	repoID = strings.ToLower(repoID)
	if x := random(n); x > 0 {
		repoID += "-"
		for n := 0; n < x; n++ {
			digit := "0123456789"
			repoID += string(digit[random([]int{0, 9})])
		}
	}
	return repoID
}

// The genChecksum returns random sha256 checksum
func genChecksum() string {
	b := genWords([]int{10, 20})
	hash := sha256.New()
	sum := hash.Sum([]byte(b))
	return hex.EncodeToString(sum)
}

func genArtifactID() string {
	switch R([]string{"semver", "hash", "uuid", "ulid"}) {
	case "hash":
		return genChecksum()
	case "uuid":
		return uuid.New().String()
	case "ulid":
		return ulid.Make().String()
	}
	return genSemver()
}

func genSemver() string {
	switch R([]string{"x.x", "x.x.x", "x.x.x.x", "x.x.x-x.x"}) {
	case "x.x":
		return fmt.Sprintf("%v.%v", random([]int{0, 20}), random([]int{0, 100}))
	case "x.x.x":
		return fmt.Sprintf("%v.%v.%v", random([]int{0, 20}), random([]int{0, 50}), random([]int{0, 200}))
	case "x.x.x.x":
		return fmt.Sprintf("%v.%v.%v.%v", random([]int{0, 50}), random([]int{0, 100}), random([]int{0, 200}), random([]int{0, 100000}))
	case "x.x.x-x.x":
		return fmt.Sprintf("%v.%v.%v-%v.%v", random([]int{0, 20}), random([]int{0, 50}), random([]int{0, 300}), R([]string{"alpha", "beta", "gamma"}), random([]int{0, 1000}))
	}

	return ""
}

var gen = loremipsum.New()
var fakeStorage = &FakeStorage{}

type FakeStorage struct {
}

func (*FakeStorage) NewArtifact(*models.Repo, models.ArtifactID, []string) (models.ArtifactID, int64, int64, error) {
	panic("not expected to be called atm!!!")
}
func (*FakeStorage) GetArtifactFiles(models.RepoID, models.ArtifactID) ([]*models.File, error) {
	files := []*models.File{}
	for x := 0; x < random(numFiles); x++ {
		file := &models.File{
			Name:  genFileName(3),
			Size:  types.Size(random([]int{128, 150000000})),
			State: vo.ArtifactState(R([]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1})),
		}
		files = append(files, file)
	}

	return files, nil
}

func genFileName(n int) string {
	a := []string{}
	for x := 0; x < random([]int{1, n}); x++ {
		a = append(a, strings.ReplaceAll(genWords([]int{1, 3}), " ", "_"))
	}

	file := strings.Join(a, string(filepath.Separator)) + "." + R([]string{"bin", "txt", "srec", "jar", "tar.gz", "html", "iso"})
	return file
}
