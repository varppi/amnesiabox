package server

import (
	"amnesiabox/internal/utils"
	"crypto/rand"
	"mime/multipart"
	"net/http"
	"regexp"
	"runtime"
	"strings"

	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"github.com/psanford/memfs"
)

var (
	nameRegex = regexp.MustCompile("^[a-zA-Z0-9]{1,50}$")
	adminKey  = utils.RandomHash()
)

func siteHandler(c *gin.Context) {
	start := strings.TrimLeft(c.Request.URL.Path, "/sites/")
	name := strings.Split(start, "/")[0]
	if _, exists := sites[name]; !exists || sites[name] == nil {
		c.String(404, "404 page not found")
		return
	}
	path := strings.TrimLeft(start, name)
	if path == "" {
		c.Redirect(301, "/sites/"+name+"/")
		return
	}
	if strings.Contains(path, "\\") {
		c.Redirect(301, "/sites/"+name+strings.ReplaceAll(path, "\\", "/"))
	}
	c.Request.URL.Path = path
	http.FileServerFS(sites[name]).ServeHTTP(c.Writer, c.Request)
}

func logOut(c *gin.Context) {
	c.SetCookie("session", "", 60*5, "/", "", true, true)
	c.Redirect(301, "/")
}

func deleteSite(c *gin.Context) {
	var site string
	var authed bool
	if site, authed = auth(c); !authed {
		forbidden(c)
		return
	}

	sites[site] = nil
	delete(sites, site)
	key, _ := c.Cookie("session")
	delete(keys, key)
	c.SetCookie("session", "", 60*5, "/", "", true, true)

	serverTemplates.ExecuteTemplate(c.Writer, "deleted.html", gin.H{"site": site, "loggedin": loggedIn(c)})
}

func updateSite(c *gin.Context) {
	var site string
	var authed bool
	if site, authed = auth(c); !authed {
		forbidden(c)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		serverError(c)
		return
	}
	store(c, site, file)

	serverTemplates.ExecuteTemplate(c.Writer, "dashboard.html", gin.H{"site": site, "state": 2, "loggedin": loggedIn(c)})
}

func dashboard(c *gin.Context) {
	var site string
	var authed bool
	if site, authed = auth(c); !authed {
		forbidden(c)
		return
	}

	key, _ := c.Cookie("session")
	if key == utils.Sha256(adminKey) {
		c.Redirect(301, "/admin")
		return
	}

	serverTemplates.ExecuteTemplate(c.Writer, "dashboard.html", gin.H{"site": site, "state": 1, "loggedin": loggedIn(c)})
}

func adminDelete(c *gin.Context) {
	session, _ := c.Cookie("session")
	if session != utils.Sha256(adminKey) {
		forbidden(c)
		return
	}

	c.Request.ParseForm()
	siteName := c.Request.FormValue("site")

	sites[siteName] = nil
	delete(sites, siteName)
	for siteKey, site := range keys {
		if site == siteName {
			delete(keys, siteKey)
		}
	}

	c.Redirect(301, "/admin")
}

func admin(c *gin.Context) {
	key, _ := c.Cookie("session")
	if key != utils.Sha256(adminKey) {
		forbidden(c)
		return
	}

	var siteNames []string
	for name := range sites {
		siteNames = append(siteNames, name)
	}
	serverTemplates.ExecuteTemplate(c.Writer, "admin.html", gin.H{"sites": siteNames, "loggedin": loggedIn(c)})
}

func login(c *gin.Context) {
	c.Request.ParseForm()
	if !solvedCaptcha(c) {
		return
	}

	key := strings.TrimSpace(c.Request.FormValue("key"))
	if key == adminKey {
		c.SetCookie("session", utils.Sha256(key), 60*5, "/", "", true, true)
		c.Redirect(301, "/admin")
	}
	if _, exists := keys[key]; !exists {
		forbidden(c)
		return
	}
	c.SetCookie("session", utils.Sha256(key), 60*5, "/", "", true, true)
	c.Redirect(301, "/dashboard")
}

func homepage(c *gin.Context) {
	var siteList []string
	for sitename, _ := range sites {
		siteList = append(siteList, sitename)
	}
	serverTemplates.ExecuteTemplate(c.Writer, "index.html",
		gin.H{"open": Config.Open,
			"sizelimit":      Config.Sizelimit,
			"sites":          siteList,
			"hidehosted":     Config.Hidehosted,
			"loggedin":       loggedIn(c),
			"disablecaptcha": Config.Disablecaptcha,
			"captcha":        captcha.New()})
}

func upload(c *gin.Context) {
	if !solvedCaptcha(c) {
		return
	}

	name := strings.TrimSpace(c.PostForm("name"))
	if !nameRegex.MatchString(name) {
		serverError(c)
		return
	}
	password := strings.TrimSpace(c.PostForm("password"))
	if !Config.Open && password != Config.Password {
		forbidden(c)
		return
	}
	if sites[name] != nil {
		serverTemplates.ExecuteTemplate(c.Writer, "alreadyexists.html", gin.H{"loggedin": loggedIn(c)})
		c.Status(401)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		serverError(c)
		return
	}

	store(c, name, file)

	randomness := make([]byte, 1024)
	rand.Read(randomness)
	key := utils.Sha256(string(randomness))
	keys[key] = name

	serverTemplates.ExecuteTemplate(c.Writer, "key.html", gin.H{"key": key, "loggedin": loggedIn(c)})
}

// Helpers
func solvedCaptcha(c *gin.Context) bool {
	if Config.Disablecaptcha {
		return true
	}
	captchaSolution := strings.TrimSpace(c.Request.FormValue("captcha"))
	captchaId := strings.TrimSpace(c.Request.FormValue("captchaid"))
	if !captcha.VerifyString(captchaId, captchaSolution) {
		c.Data(401, "text/html", []byte("invalid captcha <a href='/'>return home</a>"))
		return false
	}

	return true
}

func auth(c *gin.Context) (string, bool) {
	session, err := c.Cookie("session")
	if err != nil {
		return "", false
	}
	if session == utils.Sha256(adminKey) {
		return "", true
	}
	var site string
	for key, _site := range keys {
		if utils.Sha256(key) == session {
			site = _site
		}
	}
	if site == "" {
		return "", false
	}
	return site, true
}

// Replace site content
func store(c *gin.Context, name string, file *multipart.FileHeader) {
	if file.Size >= int64(Config.Sizelimit) {
		serverTemplates.ExecuteTemplate(c.Writer, "toobig.html", gin.H{"loggedin": loggedIn(c)})
		c.Status(400)
		return
	}

	reader, err := file.Open()
	if err != nil {
		serverError(c)
		return
	}
	buffer := make([]byte, file.Size+1024)
	n, _ := reader.Read(buffer)
	buffer = buffer[:n]

	// Unzip and store it in memory
	if sites[name] != nil {
		sites[name] = nil
		runtime.GC()
	}
	sites[name] = memfs.New()
	err = utils.Unzip(buffer, sites[name])
	if err != nil {
		serverError(c)
		return
	}
}

func loggedIn(c *gin.Context) bool {
	session, _ := c.Cookie("session")
	return session != ""
}

func forbidden(c *gin.Context) {
	serverTemplates.ExecuteTemplate(c.Writer, "forbidden.html", gin.H{"loggedin": loggedIn(c)})
	c.Status(401)
}

func serverError(c *gin.Context) {
	serverTemplates.ExecuteTemplate(c.Writer, "internalerror.html", gin.H{"loggedin": loggedIn(c)})
	c.Status(500)
}
