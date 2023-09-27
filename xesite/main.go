package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"

	"github.com/donatj/hmacsig"
	"github.com/facebookgo/flagenv"
	"github.com/go-git/go-git/v5"
	_ "github.com/joho/godotenv/autoload"
	"tailscale.com/tsweb"
	"xeiaso.net/v4/internal"
	"xeiaso.net/v4/internal/lume"
)

var (
	bind         = flag.String("bind", ":3000", "Port to listen on")
	devel        = flag.Bool("devel", false, "Enable development mode")
	gitBranch    = flag.String("git-branch", "main", "Git branch to clone")
	gitRepo      = flag.String("git-repo", "https://github.com/Xe/site", "Git repository to clone")
	githubSecret = flag.String("github-secret", "", "GitHub secret to use for webhooks")
	siteURL      = flag.String("site-url", "https://xeiaso.net/", "URL to use for the site")
)

func main() {
	flagenv.Parse()
	flag.Parse()
	internal.Slog()

	ctx := context.Background()

	ln, err := net.Listen("tcp", *bind)
	if err != nil {
		log.Fatal(err)
	}

	pc, err := NewPatreonClient()
	if err != nil {
		slog.Error("can't create patreon client", "err", err)
	}

	fs, err := lume.New(ctx, &lume.Options{
		Branch:        *gitBranch,
		Repo:          *gitRepo,
		StaticSiteDir: "lume",
		URL:           *siteURL,
		Development:   *devel,
		PatreonClient: pc,
	})
	if err != nil {
		log.Fatal(err)
	}

	defer fs.Close()

	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(fs)))
	mux.HandleFunc("/metrics", tsweb.VarzHandler)

	mux.HandleFunc("/blog.atom", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/blog/index.rss", http.StatusMovedPermanently)
	})

	// NOTE(Xe): Had to rename this page because of a Lume/Go embed bug.
	mux.HandleFunc(`/blog/%F0%9F%A5%BA`, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/blog/xn--ts9h/", http.StatusMovedPermanently)
	})

	if *devel {
		mux.HandleFunc("/.within/hook/github", func(w http.ResponseWriter, r *http.Request) {
			if err := fs.Update(r.Context()); err != nil {
				if err == git.NoErrAlreadyUpToDate {
					w.WriteHeader(http.StatusOK)
					fmt.Fprintln(w, "already up to date")
					return
				}
				log.Println(err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		})
	} else {
		gh := &GitHubWebhook{fs: fs}
		s := hmacsig.Handler256(gh, *githubSecret)
		mux.Handle("/.within/hook/github", s)
	}

	mux.Handle("/.within/hook/patreon", &PatreonWebhook{fs: fs})

	var h http.Handler = mux
	h = internal.ClackSet(fs.Clacks()).Middleware(h)
	h = internal.CacheHeader(h)

	slog.Info("starting server", "bind", *bind)
	log.Fatal(http.Serve(ln, h))
}
