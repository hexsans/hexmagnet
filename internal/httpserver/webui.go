package httpserver

import (
	"errors"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	webuife "github.com/hexsans/hexmagnet/webui"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type WebUIParams struct {
	fx.In
	Logger *zap.SugaredLogger
}

type WebUIResult struct {
	fx.Out
	Option Option `group:"http_server_options"`
}

func NewWebUI(p WebUIParams) WebUIResult {
	return WebUIResult{
		Option: &builder{
			logger: p.Logger.Named("webui"),
		},
	}
}

type builder struct {
	logger *zap.SugaredLogger
}

func (*builder) Key() string {
	return "webui"
}

func (b *builder) Apply(e *gin.Engine) error {
	webuiFS := webuife.StaticFS()

	appRoot, appRootErr := fs.Sub(webuiFS, "dist")
	if appRootErr != nil {
		b.logger.Errorf(
			"the webui app root directory is missing; run `npm run build` within the `webui` folder: %v",
			appRootErr)

		return nil
	}

	fileServer := http.FileServer(wrappedFs{http.FS(appRoot)})

	e.GET("/", gin.WrapH(fileServer))
	e.NoRoute(func(c *gin.Context) {
		if c.Request.Method == http.MethodGet {
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}

		b.logger.Warnw("no route", "path", c.Request.URL.Path)
		c.Status(http.StatusNotFound)
	})

	return nil
}

type wrappedFs struct {
	http.FileSystem
}

func (w wrappedFs) Open(name string) (http.File, error) {
	f, err := w.FileSystem.Open(name)
	if err != nil && errors.Is(err, fs.ErrNotExist) {
		return w.FileSystem.Open("/index.html")
	}

	return f, err
}
