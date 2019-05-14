package cmd

import (
	"context"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gitlab.com/olaris/olaris-server/ffmpeg"
	"gitlab.com/olaris/olaris-server/helpers"
	"gitlab.com/olaris/olaris-server/metadata"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/react"
	"gitlab.com/olaris/olaris-server/streaming"
	"net/http"
	"net/http/pprof" // For Profiling
	"os"
	"os/signal"
	"time"
)

var port int
var dbLog bool
var verbose bool

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the olaris server",
	Run: func(cmd *cobra.Command, args []string) {
		r := mux.NewRouter()

		r.PathPrefix("/debug/pprof/").HandlerFunc(pprof.Index)
		r.PathPrefix("/s").Handler(http.StripPrefix("/s", streaming.GetHandler()))
		defer streaming.Cleanup()

		mctx := app.NewMDContext(helpers.MetadataConfigPath(), dbLog, verbose)
		defer mctx.Db.Close()

		// This is just to make sure that no temp files stay behind in case the
		// garbage collection below didn't work properly for some reason.
		// This is also relevant during development because the realize auto-reload
		// tool doesn't properly send SIGTERM.
		ffmpeg.CleanTranscodingCache()
		// TODO(Leon Handreke): Find a better way to do this, maybe a global flag?
		streaming.FeedbackUrlPort = port

		appRoute := r.PathPrefix("/app").
			Handler(http.StripPrefix("/app", react.GetHandler())).
			Name("app")

		appURL, _ := appRoute.URL()
		r.Path("/").Handler(http.RedirectHandler(appURL.Path, http.StatusMovedPermanently))

		r.PathPrefix("/m").Handler(http.StripPrefix("/m", metadata.GetHandler(mctx)))

		handler := handlers.LoggingHandler(os.Stdout, r)

		log.Infoln("binding on port", port)
		srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: handler}
		go func() {
			if err := srv.ListenAndServe(); err != nil {
				log.WithFields(log.Fields{"error": err}).Fatal("Error starting server.")
			}
		}()

		stopChan := make(chan os.Signal)
		signal.Notify(stopChan, os.Interrupt)
		signal.Notify(stopChan, os.Kill)

		// Wait for termination signal
		<-stopChan
		log.Println("Shutting down...")

		//	mctx.ExitChan <- 1
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)

		streaming.Cleanup()
	},
}

func init() {
	serveCmd.Flags().IntVarP(&port, "port", "p", 8080, "http port")
	serveCmd.Flags().BoolVarP(&verbose, "verbose", "v", true, "verbose logging")
	serveCmd.Flags().BoolVar(&dbLog, "db-log", true, "sets whether the database should log queries")
	rootCmd.AddCommand(serveCmd)
}
